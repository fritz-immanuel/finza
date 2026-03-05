# Money Tracker Telegram Bot (Go)

Production-ready Telegram personal money tracker bot in Go 1.22+ with:
- Telegram Bot API (`github.com/go-telegram-bot-api/telegram-bot-api/v5`)
- MySQL storage (`github.com/go-sql-driver/mysql`)
- Rule parser + Gemini Flash fallback (`github.com/google/generative-ai-go/genai`)

## Local Setup

1. Create MySQL DB:

```sql
CREATE DATABASE finza;
```

2. Copy environment file and fill values:

```bash
cp .env.example .env
```

3. Put your values in `.env` (source of truth). The app reads `.env` first and only falls back to process env vars if a key is missing in `.env`.

4. Install dependencies and run:

```bash
go mod tidy
go run ./cmd/bot
```

At startup, the bot runs `schema.sql` automatically.

## Webhook Setup

Set:
- `BOT_MODE=webhook`
- `BOT_WEBHOOK_URL=https://yourdomain.com`
- `BOT_WEBHOOK_ADDR=:8443`
- `BOT_WEBHOOK_PATH=/webhook`
- `BOT_WEBHOOK_SECRET=<secret>`

Run bot behind HTTPS (reverse proxy or direct TLS terminator). The bot registers webhook using `BOT_WEBHOOK_URL + BOT_WEBHOOK_PATH`.

## Environment Variables

- `TELEGRAM_BOT_TOKEN` (required): Telegram bot token
- `MYSQL_DSN` (required): MySQL DSN with `parseTime=true&loc=UTC`
- `GEMINI_API_KEY`: Gemini key for LLM fallback parser
- `GEMINI_MODEL`: Gemini model name (default `gemini-2.0-flash`)
- `LLM_CONFIDENCE_THRESHOLD`: Confidence threshold (default `0.60`)
- `BOT_MODE`: `polling` or `webhook`
- `BOT_WEBHOOK_URL`: Public base URL (webhook mode)
- `BOT_WEBHOOK_ADDR`: Local listen address (webhook mode)
- `BOT_WEBHOOK_PATH`: Webhook path (webhook mode)
- `BOT_WEBHOOK_SECRET`: Webhook secret (webhook mode)
- `BOT_WORKERS`: Worker goroutines count
- `BOT_QUEUE_SIZE`: Update queue buffer size
- `BOT_RATE_LIMIT_RPS`: Per-user refill tokens/second
- `BOT_RATE_LIMIT_BURST`: Per-user burst tokens
- `BOT_DEBUG`: Enable debug log level

## Data Model

### `users`
Stores Telegram identity and user timezone.

### `raw_messages`
Stores all inbound raw text before parsing (auditability).

### `entries`
Stores normalized financial records:
- `amount`: base units (`IDR/JPY` integer, `USD/EUR/SGD/GBP` cents)
- `type`: `expense|income|transfer|unknown`
- `confidence`: parser confidence score

All DB timestamps are stored in UTC.

## Swapping LLM Parser

Current parser chain:
1. `parser/rule.RuleParser`
2. `parser/llm.GeminiParser` fallback (if rule confidence < threshold)

To swap LLM parser:
1. Implement `domain.Parser`
2. Replace Gemini parser construction in `cmd/bot/main.go`
3. Keep `llm.ChainParser` as-is (or replace with your own chain)

## Commands

- `/start`
- `/help`
- `/add <text>`
- `/last [n]`
- `/today`
- `/week`
- `/month`
- `/report YYYY-MM`
- `/categories YYYY-MM`
- `/delete <id>`
- `/export`
- `/timezone <tz>`

## Tests

Run:

```bash
go test ./...
```
