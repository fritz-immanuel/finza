package telegram

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/yourusername/moneytracker/app"
	"github.com/yourusername/moneytracker/domain"
	reportpkg "github.com/yourusername/moneytracker/report"
)

type Handler struct {
	bot            domain.BotClient
	store          domain.Store
	expenseService *app.ExpenseService
	reportService  *app.ReportService
	logger         *slog.Logger

	onStart func(context.Context, tgbotapi.Update) error
	onHelp  func(context.Context, tgbotapi.Update) error
}

func NewHandler(bot domain.BotClient, store domain.Store, expenseService *app.ExpenseService, reportService *app.ReportService, logger *slog.Logger) *Handler {
	h := &Handler{
		bot:            bot,
		store:          store,
		expenseService: expenseService,
		reportService:  reportService,
		logger:         logger,
	}
	h.onStart = h.startHandler
	h.onHelp = h.helpHandler
	return h
}

func (h *Handler) HandleUpdate(ctx context.Context, update tgbotapi.Update) error {
	if update.CallbackQuery != nil {
		return h.handleCallback(ctx, update)
	}
	if update.Message == nil {
		return nil
	}

	msg := update.Message
	if msg.IsCommand() {
		return h.handleCommand(ctx, update)
	}

	if strings.TrimSpace(msg.Text) == "" {
		return nil
	}

	return h.handleFreeText(ctx, msg.From, msg.Chat.ID, msg.MessageID, msg.Text)
}

func (h *Handler) handleCommand(ctx context.Context, update tgbotapi.Update) error {
	msg := update.Message
	command := msg.Command()

	switch command {
	case "start":
		return h.onStart(ctx, update)
	case "help":
		return h.onHelp(ctx, update)
	case "add":
		text := strings.TrimSpace(msg.CommandArguments())
		if text == "" {
			return h.sendMessage(msg.Chat.ID, "Usage: /add <transaction text>")
		}
		return h.handleFreeText(ctx, msg.From, msg.Chat.ID, msg.MessageID, text)
	case "last":
		return h.lastHandler(ctx, update)
	case "today":
		return h.summaryHandler(ctx, update, "Today")
	case "week":
		return h.summaryHandler(ctx, update, "Week")
	case "month":
		return h.summaryHandler(ctx, update, "Month")
	case "report":
		return h.reportMonthHandler(ctx, update)
	case "categories":
		return h.categoriesHandler(ctx, update)
	case "delete":
		return h.deleteHandler(ctx, update)
	case "export":
		return h.exportHandler(ctx, update)
	case "timezone":
		return h.timezoneHandler(ctx, update)
	default:
		return h.sendMessage(msg.Chat.ID, "Unknown command. Use /help.")
	}
}

func (h *Handler) startHandler(ctx context.Context, update tgbotapi.Update) error {
	msg := update.Message
	user := domain.User{
		ID:        msg.From.ID,
		Username:  msg.From.UserName,
		FirstName: msg.From.FirstName,
		LastName:  msg.From.LastName,
		Timezone:  "Asia/Jakarta",
	}
	if err := h.store.UpsertUser(ctx, user); err != nil {
		return err
	}
	text := "Welcome to Money Tracker Bot.\nSend messages like: coffee 25k, salary Rp 5jt, transfer 100k.\nUse /help for commands."
	return h.sendMessage(msg.Chat.ID, text)
}

func (h *Handler) helpHandler(_ context.Context, update tgbotapi.Update) error {
	text := strings.Join([]string{
		"/start - Welcome and usage",
		"/help - Show commands",
		"/add <text> - Add transaction",
		"/last [n] - Last n entries",
		"/today - Today summary",
		"/week - Week summary",
		"/month - Month summary",
		"/report YYYY-MM - Month summary",
		"/categories YYYY-MM - Category totals",
		"/delete <id> - Delete entry",
		"/export - Export CSV",
		"/timezone <tz> - Set timezone",
	}, "\n")
	return h.sendMessage(update.Message.Chat.ID, text)
}

func (h *Handler) handleFreeText(ctx context.Context, from *tgbotapi.User, chatID int64, messageID int, text string) error {
	user := domain.User{
		ID:        from.ID,
		Username:  from.UserName,
		FirstName: from.FirstName,
		LastName:  from.LastName,
		Timezone:  userTimezone(ctx, h.store, from.ID),
	}

	result, err := h.expenseService.ProcessMessage(ctx, user, chatID, messageID, text)
	if err != nil {
		return err
	}

	if !result.Financial {
		return h.sendMessage(chatID, "I couldn't identify a transaction in that message. Try something like: 'coffee 25k' or 'salary Rp 5jt'.")
	}

	if !result.RequiresConfirm {
		return h.sendMessage(chatID, confirmedText(result.Entry))
	}

	body := fmt.Sprintf("Saved (unconfirmed):\n%s %s %s\n\nIs this an expense or income?",
		result.Entry.Description,
		reportpkg.FormatAmount(result.Entry.Currency, result.Entry.Amount),
		result.Entry.Currency,
	)

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💸 Expense", fmt.Sprintf("confirm:%d:expense", result.Entry.ID)),
			tgbotapi.NewInlineKeyboardButtonData("💰 Income", fmt.Sprintf("confirm:%d:income", result.Entry.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Transfer", fmt.Sprintf("confirm:%d:transfer", result.Entry.ID)),
			tgbotapi.NewInlineKeyboardButtonData("🚫 Ignore", fmt.Sprintf("confirm:%d:ignore", result.Entry.ID)),
		),
	)

	m := tgbotapi.NewMessage(chatID, body)
	m.ReplyMarkup = kb
	_, err = h.bot.Send(m)
	return err
}

func (h *Handler) lastHandler(ctx context.Context, update tgbotapi.Update) error {
	n := 5
	arg := strings.TrimSpace(update.Message.CommandArguments())
	if arg != "" {
		v, err := strconv.Atoi(arg)
		if err != nil {
			return h.sendMessage(update.Message.Chat.ID, "Usage: /last [n]")
		}
		if v < 1 {
			v = 1
		}
		if v > 20 {
			v = 20
		}
		n = v
	}

	entries, err := h.reportService.Last(ctx, update.Message.From.ID, n, time.UTC)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return h.sendMessage(update.Message.Chat.ID, "No entries yet.")
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("📋 Last %d entries:\n\n", n))
	for i, e := range entries {
		icon := "❓"
		if e.Type == domain.EntryTypeExpense {
			icon = "💸"
		} else if e.Type == domain.EntryTypeIncome {
			icon = "💰"
		} else if e.Type == domain.EntryTypeTransfer {
			icon = "🔄"
		}

		b.WriteString(fmt.Sprintf("%d. [%d] %s %s — %s (%s) · %s\n",
			i+1,
			e.ID,
			icon,
			e.Description,
			reportpkg.FormatAmount(e.Currency, e.Amount),
			e.Category,
			e.Timestamp.Format("15:04"),
		))
	}

	return h.sendMessage(update.Message.Chat.ID, strings.TrimSpace(b.String()))
}

func (h *Handler) summaryHandler(ctx context.Context, update tgbotapi.Update, mode string) error {
	user, err := h.store.GetUser(ctx, update.Message.From.ID)
	if err != nil {
		user.Timezone = "Asia/Jakarta"
	}
	loc, err := time.LoadLocation(user.Timezone)
	if err != nil {
		loc = time.FixedZone("UTC+7", 7*3600)
	}

	now := time.Now().In(loc)
	var from, to time.Time
	title := mode

	switch mode {
	case "Today":
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		to = from.AddDate(0, 0, 1)
	case "Week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -(weekday - 1))
		to = from.AddDate(0, 0, 7)
	default:
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		to = from.AddDate(0, 1, 0)
	}

	summary, err := h.reportService.Summary(ctx, update.Message.From.ID, title, from, to, loc)
	if err != nil {
		return err
	}
	return h.sendMessage(update.Message.Chat.ID, summary)
}

func (h *Handler) reportMonthHandler(ctx context.Context, update tgbotapi.Update) error {
	arg := strings.TrimSpace(update.Message.CommandArguments())
	t, err := time.Parse("2006-01", arg)
	if err != nil {
		return h.sendMessage(update.Message.Chat.ID, "Usage: /report YYYY-MM")
	}

	from := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	text, err := h.reportService.Summary(ctx, update.Message.From.ID, "Report", from, to, time.UTC)
	if err != nil {
		return err
	}
	return h.sendMessage(update.Message.Chat.ID, text)
}

func (h *Handler) categoriesHandler(ctx context.Context, update tgbotapi.Update) error {
	arg := strings.TrimSpace(update.Message.CommandArguments())
	t, err := time.Parse("2006-01", arg)
	if err != nil {
		return h.sendMessage(update.Message.Chat.ID, "Usage: /categories YYYY-MM")
	}

	from := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	totals, err := h.reportService.Categories(ctx, update.Message.From.ID, from, to)
	if err != nil {
		return err
	}

	if len(totals) == 0 {
		return h.sendMessage(update.Message.Chat.ID, "No category data for that month.")
	}

	var b strings.Builder
	b.WriteString("📁 Category breakdown:\n")
	for _, cat := range reportpkg.SortedCategories(totals) {
		b.WriteString(fmt.Sprintf("- %s: %s\n", cat, reportpkg.FormatAmount("IDR", totals[cat])))
	}
	return h.sendMessage(update.Message.Chat.ID, strings.TrimSpace(b.String()))
}

func (h *Handler) deleteHandler(ctx context.Context, update tgbotapi.Update) error {
	arg := strings.TrimSpace(update.Message.CommandArguments())
	if arg == "" {
		return h.showDeleteMenu(ctx, update.Message.Chat.ID, update.Message.From.ID, 0)
	}

	id, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return h.sendMessage(update.Message.Chat.ID, "Usage: /delete <id> or /delete")
	}

	if err := h.expenseService.DeleteEntry(ctx, update.Message.From.ID, id); err != nil {
		return h.sendMessage(update.Message.Chat.ID, "Entry not found.")
	}
	return h.sendMessage(update.Message.Chat.ID, "Entry deleted.")
}

func (h *Handler) showDeleteMenu(ctx context.Context, chatID, userID int64, page int) error {
	const (
		fetchLimit = 50
		pageSize   = 5
	)

	entries, err := h.store.GetLastEntries(ctx, userID, fetchLimit)
	if err != nil {
		return err
	}

	entriesCount := len(entries)

	if entriesCount == 0 {
		return h.sendMessage(chatID, "No entries available to delete.")
	}

	totalPages := (entriesCount + pageSize - 1) / pageSize
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	start := page * pageSize
	end := start + pageSize
	if end > entriesCount {
		end = entriesCount
	}
	slice := entries[start:end]

	var b strings.Builder
	b.WriteString(fmt.Sprintf("🗑 Select entry to delete (Page %d/%d):\n\n", page+1, totalPages))
	for i, e := range slice {
		b.WriteString(fmt.Sprintf("%d. %s — %s\n",
			i+1,
			truncateText(e.Description, 24),
			reportpkg.FormatAmount(e.Currency, e.Amount),
		))
	}

	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(slice)+1)
	for _, e := range slice {
		label := fmt.Sprintf("🗑 %s", truncateText(e.Description, 18))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("delgo:%d:%d", e.ID, page)),
		))
	}

	nav := make([]tgbotapi.InlineKeyboardButton, 0, 3)
	if page > 0 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅️ Prev", fmt.Sprintf("delnav:%d", page-1)))
	}
	if page < totalPages-1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("Next ➡️", fmt.Sprintf("delnav:%d", page+1)))
	}
	nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("Cancel", "delcancel"))
	rows = append(rows, nav)

	msg := tgbotapi.NewMessage(chatID, strings.TrimSpace(b.String()))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	_, err = h.bot.Send(msg)
	return err
}

func (h *Handler) exportHandler(ctx context.Context, update tgbotapi.Update) error {
	data, err := h.reportService.ExportCSV(ctx, update.Message.From.ID)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("transactions_%s.csv", time.Now().Format("2006-01"))
	file := tgbotapi.FileBytes{Name: filename, Bytes: data}
	doc := tgbotapi.NewDocument(update.Message.Chat.ID, file)
	_, err = h.bot.Send(doc)
	return err
}

func (h *Handler) timezoneHandler(ctx context.Context, update tgbotapi.Update) error {
	tz := strings.TrimSpace(update.Message.CommandArguments())
	if tz == "" {
		return h.sendMessage(update.Message.Chat.ID, "Usage: /timezone <Area/City>")
	}

	if _, err := time.LoadLocation(tz); err != nil {
		return h.sendMessage(update.Message.Chat.ID, "Invalid timezone.")
	}

	user, err := h.store.GetUser(ctx, update.Message.From.ID)
	if err != nil {
		user = domain.User{ID: update.Message.From.ID}
	}
	user.Username = update.Message.From.UserName
	user.FirstName = update.Message.From.FirstName
	user.LastName = update.Message.From.LastName
	user.Timezone = tz

	if err := h.store.UpsertUser(ctx, user); err != nil {
		return err
	}

	return h.sendMessage(update.Message.Chat.ID, "Timezone updated to "+tz)
}

func (h *Handler) sendMessage(chatID int64, text string) error {
	var err error
	for i := 0; i < 3; i++ {
		_, err = h.bot.Send(tgbotapi.NewMessage(chatID, text))
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(100*(1<<i)) * time.Millisecond)
	}
	return err
}

func (h *Handler) editMessage(chatID int64, messageID int, text string) error {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	_, err := h.bot.Request(edit)
	return err
}

func userTimezone(ctx context.Context, store domain.Store, userID int64) string {
	user, err := store.GetUser(ctx, userID)
	if err != nil || user.Timezone == "" {
		return "Asia/Jakarta"
	}
	return user.Timezone
}

func (h *Handler) sendError(chatID int64, err error) {
	h.logger.Error("handler error", "error", err)
	_ = h.sendMessage(chatID, "Something went wrong, please try again.")
}

func asCSVBuffer(data []byte) *bytes.Reader {
	return bytes.NewReader(data)
}

func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}
