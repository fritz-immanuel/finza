# Code Flow Documentation

## 1. Startup Flow

Entry point: `cmd/bot/main.go`

1. Load config from `.env` using `config.Load()` (with `os.Getenv` fallback per key).
2. Initialize structured logger (`internal/logger`).
3. Open MySQL connection and verify `Ping()`.
4. Run DB migrations from `schema.sql` (`store/mysql/migrations.go`).
5. Build dependencies:
   - Store: `store/mysql.MySQLStore`
   - Rule parser: `parser/rule.RuleParser`
   - LLM parser: `parser/llm.GeminiParser` (only if `GEMINI_API_KEY` set)
   - Chain parser: `parser/llm.ChainParser`
   - Services: `app.ExpenseService`, `app.ReportService`
   - Telegram handler + runner
6. Start bot in `polling` or `webhook` mode.

## 2. Update Processing Flow

Core runtime: `transport/telegram/bot.go`

1. Updates are received from Telegram (polling channel or webhook HTTP endpoint).
2. Each update is submitted to a worker pool (`internal/workerpool`).
3. Per-user lock (`internal/userlock`) ensures sequential processing for the same user.
4. Middleware chain wraps handler execution:
   - Recovery middleware
   - Rate-limit middleware (`internal/ratelimit`)
   - Logger middleware (`log/slog`)
5. Final dispatch goes to `Handler.HandleUpdate()`.

## 3. Message Routing Flow

Router: `transport/telegram/handler.go`

- If callback query: route to `handleCallback()`.
- If command: route to `handleCommand()`.
- If plain text: route to `handleFreeText()`.

## 4. Plain Text Transaction Flow

Path: `handleFreeText()` -> `ExpenseService.ProcessMessage()`

1. Upsert user in `users` table.
2. Save raw text in `raw_messages` table.
3. Parse text with `ChainParser`:
   - Try rule parser first.
   - If confidence below threshold, fallback to Gemini parser.
   - If Gemini fails, fallback to rule result.
4. If not financial: send hint message.
5. If financial:
   - Save entry to `entries`.
   - If confidence >= 0.75 and type known: send immediate confirmation.
   - Else save as `unknown` and ask for inline confirmation keyboard.

## 5. Callback Confirmation Flow

Handler: `transport/telegram/callback.go`

Callback format: `confirm:<entryID>:<type>`

- `expense|income|transfer`:
  1. Update entry type in DB.
  2. Edit bot message to confirmed summary.
- `ignore`:
  1. Delete entry in DB.
  2. Edit message to `Entry discarded.`

## 6. Command Flow

In `handleCommand()`:

- `/start`, `/help`: static/help responses + user upsert.
- `/add <text>`: force parse path (same as free text).
- `/last [n]`: fetch latest entries and format list.
- `/today`, `/week`, `/month`, `/report YYYY-MM`: date range summary via report service.
- `/categories YYYY-MM`: category totals.
- `/delete <id>`: ownership check + delete.
- `/export`: fetch entries, generate CSV, send document.
- `/timezone <tz>`: validate and persist timezone.

## 7. Parser Flow

### Rule Parser (`parser/rule`)

1. Detect currency (default `IDR`).
2. Extract amount token and normalize (`k`, `rb`, `jt`, `m`, separators, cents rules).
3. Score keywords (income/expense/transfer).
4. Infer entry type and category.
5. Set confidence:
   - 0.95: currency + keyword + amount
   - 0.75: keyword + amount
   - 0.50: amount only
   - 0.00: no amount

### Gemini Parser (`parser/llm`)

1. Build structured prompt.
2. Call `GEMINI_MODEL` with retries/backoff.
3. Extract JSON object from response.
4. Validate `is_financial` and map JSON to `domain.Entry`.

## 8. Report Flow

Service: `app/report.go`

1. Query entries by date range from store.
2. Aggregate totals in `report/aggregator.go`.
3. Format message output in `report/formatter.go`.
4. For export, generate CSV in `report/csv.go`.

## 9. Data Access Flow

Store implementation: `store/mysql/store.go`

- Uses query constants in `store/mysql/queries.go`.
- Wraps DB errors with operation context.
- Converts SQL rows into domain structs.

## 10. Time & Timezone Handling

- DB timestamps are stored in UTC.
- User timezone is stored in `users.timezone`.
- Display/report periods are converted using the user timezone.

## 11. Reliability Notes

- Startup exits fast if DB unavailable.
- Telegram send retries with backoff.
- Panic recovery prevents process crash for bad updates.
- Per-user lock prevents race conditions on same user data.
- Rate limiter protects bot from burst spam.
