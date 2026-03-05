package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/yourusername/moneytracker/domain"
	"google.golang.org/api/option"
)

type GeminiParser struct {
	client *genai.Client
	logger *slog.Logger
	model  string
}

type llmResponse struct {
	Type        string  `json:"type"`
	Amount      int64   `json:"amount"`
	Currency    string  `json:"currency"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence"`
	IsFinancial bool    `json:"is_financial"`
}

func NewGeminiParser(ctx context.Context, apiKey, model string, logger *slog.Logger) (*GeminiParser, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("genai.NewClient: %w", err)
	}
	if strings.TrimSpace(model) == "" {
		model = "gemini-2.0-flash"
	}
	return &GeminiParser{client: client, logger: logger, model: model}, nil
}

func (p *GeminiParser) Parse(ctx context.Context, text string) (domain.Entry, bool, float64, error) {
	prompt, err := BuildPrompt(PromptInput{Text: text})
	if err != nil {
		return domain.Entry{}, false, 0, fmt.Errorf("BuildPrompt: %w", err)
	}

	model := p.client.GenerativeModel(p.model)

	var raw string
	for attempt := 0; attempt < 3; attempt++ {
		resp, callErr := model.GenerateContent(ctx, genai.Text(prompt))
		if callErr != nil {
			if attempt == 2 {
				p.logger.Error("gemini call failed", "error", callErr)
				return domain.Entry{}, false, 0, callErr
			}
			time.Sleep(time.Duration(100*(1<<attempt)) * time.Millisecond)
			continue
		}
		raw = flattenResponse(resp)
		if raw != "" {
			break
		}
	}

	raw = extractJSONObject(raw)
	if raw == "" {
		p.logger.Warn("gemini returned empty response")
		return domain.Entry{}, false, 0, nil
	}

	var parsed llmResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		p.logger.Error("gemini json parse failed", "error", err, "raw", raw)
		return domain.Entry{}, false, 0, nil
	}

	if !parsed.IsFinancial {
		return domain.Entry{}, false, 0, nil
	}

	entry := domain.Entry{
		Timestamp:   time.Now().UTC(),
		Amount:      parsed.Amount,
		Currency:    strings.ToUpper(strings.TrimSpace(parsed.Currency)),
		Type:        domain.EntryType(parsed.Type),
		Category:    parsed.Category,
		Description: parsed.Description,
		RawText:     text,
		Confidence:  parsed.Confidence,
	}

	return entry, true, parsed.Confidence, nil
}

func flattenResponse(resp *genai.GenerateContentResponse) string {
	if resp == nil {
		return ""
	}
	var b strings.Builder
	for _, c := range resp.Candidates {
		if c == nil || c.Content == nil {
			continue
		}
		for _, part := range c.Content.Parts {
			if t, ok := part.(genai.Text); ok {
				b.WriteString(string(t))
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func extractJSONObject(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return s[start : end+1]
}

type ChainParser struct {
	Rule      domain.Parser
	LLM       domain.Parser
	Threshold float64
}

func NewChainParser(ruleParser, llmParser domain.Parser, threshold float64) *ChainParser {
	if threshold <= 0 {
		threshold = 0.60
	}
	return &ChainParser{Rule: ruleParser, LLM: llmParser, Threshold: threshold}
}

func (p *ChainParser) Parse(ctx context.Context, text string) (domain.Entry, bool, float64, error) {
	ruleEntry, ruleOK, ruleConf, ruleErr := p.Rule.Parse(ctx, text)
	if ruleErr != nil {
		return domain.Entry{}, false, 0, ruleErr
	}
	if ruleConf >= p.Threshold || p.LLM == nil {
		return ruleEntry, ruleOK, ruleConf, nil
	}

	llmEntry, llmOK, llmConf, llmErr := p.LLM.Parse(ctx, text)
	if llmErr != nil {
		return ruleEntry, ruleOK, ruleConf, nil
	}
	return llmEntry, llmOK, llmConf, nil
}
