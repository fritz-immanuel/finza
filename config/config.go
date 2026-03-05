package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Token         string
	DSN           string
	Mode          string
	WebhookURL    string
	WebhookAddr   string
	WebhookPath   string
	WebhookSecret string
	Workers       int
	QueueSize     int
	Debug         bool
	GeminiKey     string
	GeminiModel   string
	LLMThreshold  float64
	RateRPS       float64
	RateBurst     int
}

func Load(path string) (Config, error) {
	envFile, err := readDotEnv(path)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Token:         get(envFile, "TELEGRAM_BOT_TOKEN"),
		DSN:           get(envFile, "MYSQL_DSN"),
		Mode:          getWithDefault(envFile, "BOT_MODE", "polling"),
		WebhookURL:    get(envFile, "BOT_WEBHOOK_URL"),
		WebhookAddr:   getWithDefault(envFile, "BOT_WEBHOOK_ADDR", ":8443"),
		WebhookPath:   getWithDefault(envFile, "BOT_WEBHOOK_PATH", "/webhook"),
		WebhookSecret: get(envFile, "BOT_WEBHOOK_SECRET"),
		Workers:       getInt(envFile, "BOT_WORKERS", 4),
		QueueSize:     getInt(envFile, "BOT_QUEUE_SIZE", 100),
		Debug:         getBool(envFile, "BOT_DEBUG", false),
		GeminiKey:     get(envFile, "GEMINI_API_KEY"),
		GeminiModel:   getWithDefault(envFile, "GEMINI_MODEL", "gemini-2.0-flash"),
		LLMThreshold:  getFloat(envFile, "LLM_CONFIDENCE_THRESHOLD", 0.60),
		RateRPS:       getFloat(envFile, "BOT_RATE_LIMIT_RPS", 2),
		RateBurst:     getInt(envFile, "BOT_RATE_LIMIT_BURST", 5),
	}

	if cfg.Token == "" || cfg.DSN == "" {
		return Config{}, fmt.Errorf("missing required config: TELEGRAM_BOT_TOKEN, MYSQL_DSN")
	}

	return cfg, nil
}

func readDotEnv(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	vals := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "\"")

		if key == "" {
			continue
		}
		vals[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	return vals, nil
}

func get(envFile map[string]string, key string) string {
	if v, ok := envFile[key]; ok {
		return v
	}
	return os.Getenv(key)
}

func getWithDefault(envFile map[string]string, key, d string) string {
	v := get(envFile, key)
	if v == "" {
		return d
	}
	return v
}

func getInt(envFile map[string]string, key string, d int) int {
	v := get(envFile, key)
	if v == "" {
		return d
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return d
	}
	return i
}

func getFloat(envFile map[string]string, key string, d float64) float64 {
	v := get(envFile, key)
	if v == "" {
		return d
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return d
	}
	return f
}

func getBool(envFile map[string]string, key string, d bool) bool {
	v := get(envFile, key)
	if v == "" {
		return d
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return d
	}
	return b
}
