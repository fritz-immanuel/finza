package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/yourusername/moneytracker/domain"
	reportpkg "github.com/yourusername/moneytracker/report"
)

func (h *Handler) handleCallback(ctx context.Context, update tgbotapi.Update) error {
	cb := update.CallbackQuery
	if cb == nil {
		return nil
	}

	_, _ = h.bot.Request(tgbotapi.NewCallback(cb.ID, "Processing..."))

	parts := strings.Split(cb.Data, ":")
	if len(parts) == 0 {
		return h.sendMessage(cb.Message.Chat.ID, "Invalid action.")
	}

	switch parts[0] {
	case "confirm":
		return h.handleConfirmCallback(ctx, cb, parts)
	case "delnav":
		return h.handleDeleteNavCallback(ctx, cb, parts)
	case "delgo":
		return h.handleDeleteDoCallback(ctx, cb, parts)
	case "delcancel":
		return h.editMessage(cb.Message.Chat.ID, cb.Message.MessageID, "Delete cancelled.")
	default:
		return h.sendMessage(cb.Message.Chat.ID, "Invalid action.")
	}
}

func (h *Handler) handleConfirmCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, parts []string) error {
	if len(parts) != 3 {
		return h.sendMessage(cb.Message.Chat.ID, "Invalid confirmation action.")
	}

	entryID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return h.sendMessage(cb.Message.Chat.ID, "Invalid entry ID.")
	}

	switch parts[2] {
	case string(domain.EntryTypeExpense), string(domain.EntryTypeIncome), string(domain.EntryTypeTransfer):
		if err := h.expenseService.ConfirmEntryType(ctx, entryID, domain.EntryType(parts[2])); err != nil {
			return err
		}

		entry, ok, err := h.store.GetEntryByID(ctx, entryID)
		if err != nil {
			return err
		}
		if !ok {
			return h.editMessage(cb.Message.Chat.ID, cb.Message.MessageID, "Entry not found.")
		}

		text := confirmedText(entry)
		edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, text)
		_, err = h.bot.Request(edit)
		return err
	case "ignore":
		if err := h.expenseService.DeleteEntry(ctx, cb.From.ID, entryID); err != nil {
			return err
		}
		return h.editMessage(cb.Message.Chat.ID, cb.Message.MessageID, "Entry discarded.")
	default:
		return h.sendMessage(cb.Message.Chat.ID, "Unknown action.")
	}
}

func (h *Handler) handleDeleteNavCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, parts []string) error {
	if len(parts) != 2 {
		return h.sendMessage(cb.Message.Chat.ID, "Invalid delete navigation.")
	}
	page, err := strconv.Atoi(parts[1])
	if err != nil {
		return h.sendMessage(cb.Message.Chat.ID, "Invalid page.")
	}
	return h.editDeleteMenu(ctx, cb.Message.Chat.ID, cb.Message.MessageID, cb.From.ID, page)
}

func (h *Handler) handleDeleteDoCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, parts []string) error {
	if len(parts) != 3 {
		return h.sendMessage(cb.Message.Chat.ID, "Invalid delete action.")
	}

	entryID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return h.sendMessage(cb.Message.Chat.ID, "Invalid entry ID.")
	}
	page, err := strconv.Atoi(parts[2])
	if err != nil {
		page = 0
	}

	if err := h.expenseService.DeleteEntry(ctx, cb.From.ID, entryID); err != nil {
		return h.sendMessage(cb.Message.Chat.ID, "Entry not found.")
	}
	return h.editDeleteMenu(ctx, cb.Message.Chat.ID, cb.Message.MessageID, cb.From.ID, page)
}

func (h *Handler) editDeleteMenu(ctx context.Context, chatID int64, messageID int, userID int64, page int) error {
	const (
		fetchLimit = 50
		pageSize   = 5
	)

	entries, err := h.store.GetLastEntries(ctx, userID, fetchLimit)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return h.editMessage(chatID, messageID, "No entries available to delete.")
	}

	totalPages := (len(entries) + pageSize - 1) / pageSize
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	start := page * pageSize
	end := start + pageSize
	if end > len(entries) {
		end = len(entries)
	}
	slice := entries[start:end]

	var b strings.Builder
	b.WriteString(fmt.Sprintf("🗑 Select entry to delete (Page %d/%d):\n\n", page+1, totalPages))
	for i, e := range slice {
		b.WriteString(fmt.Sprintf("%d. [%d] %s — %s\n",
			i+1,
			e.ID,
			truncateText(e.Description, 24),
			reportpkg.FormatAmount(e.Currency, e.Amount),
		))
	}

	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(slice)+1)
	for _, e := range slice {
		label := fmt.Sprintf("🗑 [%d] %s", e.ID, truncateText(e.Description, 18))
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

	edit := tgbotapi.NewEditMessageText(chatID, messageID, strings.TrimSpace(b.String()))
	edit.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
	_, err = h.bot.Request(edit)
	return err
}

func confirmedText(entry domain.Entry) string {
	return fmt.Sprintf("✅ Saved %s:\n%s\n%s\n📁 %s",
		entry.Type,
		entry.Description,
		reportpkg.FormatAmount(entry.Currency, entry.Amount),
		entry.Category,
	)
}
