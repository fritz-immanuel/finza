package llm

import (
	"bytes"
	"text/template"
	"time"
)

const promptTemplate = `You are a financial transaction parser for a personal money tracker app used primarily in Indonesia.

The user sent this message: "{{.Text}}"

Today's date: {{.Date}}
User's recent entries for context: {{.RecentEntries}}

Extract the financial transaction from the message. Respond ONLY with a valid JSON object, no markdown, no explanation:

{
  "type": "expense|income|transfer|unknown",
  "amount": <number, integer, in base currency units>,
  "currency": "<ISO 4217 code: IDR, USD, JPY, SGD, EUR, GBP, MYR, THB>",
  "category": "<inferred category string>",
  "description": "<clean short description of what the transaction is>",
  "confidence": <float between 0.0 and 1.0>,
  "is_financial": <true|false>
}

Rules:
- If the message is not financial at all, set is_financial=false and confidence=0.0
- For IDR amounts: 10jt = 10000000, 75k = 75000, 75rb = 75000
- Default currency to IDR if none is specified
- Be concise in description (max 5 words)
- Category must be one of: Food & Drink, Transport, Shopping, Utilities, Health, Housing, Entertainment, Education, Income, Transfer, General`

type PromptInput struct {
	Text          string
	Date          string
	RecentEntries string
}

func BuildPrompt(in PromptInput) (string, error) {
	if in.Date == "" {
		in.Date = time.Now().UTC().Format("2006-01-02")
	}
	if in.RecentEntries == "" {
		in.RecentEntries = "[]"
	}

	tmpl, err := template.New("gemini-prompt").Parse(promptTemplate)
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	if err := tmpl.Execute(&b, in); err != nil {
		return "", err
	}
	return b.String(), nil
}
