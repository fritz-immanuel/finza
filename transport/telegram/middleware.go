package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/yourusername/moneytracker/domain"
	"github.com/yourusername/moneytracker/internal/ratelimit"
)

type HandlerFunc func(ctx context.Context, update tgbotapi.Update) error

type Middleware func(HandlerFunc) HandlerFunc

func Chain(h HandlerFunc, m ...Middleware) HandlerFunc {
	wrapped := h
	for i := len(m) - 1; i >= 0; i-- {
		wrapped = m[i](wrapped)
	}
	return wrapped
}

func LoggerMiddleware(logger *slog.Logger, handlerName string) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, update tgbotapi.Update) error {
			start := time.Now()
			err := next(ctx, update)
			userID, chatID := extractIDs(update)
			logger.Info("update processed",
				"update_id", update.UpdateID,
				"user_id", userID,
				"chat_id", chatID,
				"handler", handlerName,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return err
		}
	}
}

func RecoveryMiddleware(logger *slog.Logger, bot domain.BotClient) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, update tgbotapi.Update) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("panic recovered", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
					_, _ = bot.Send(tgbotapi.NewMessage(chatID(update), "Something went wrong, please try again."))
					err = nil
				}
			}()
			return next(ctx, update)
		}
	}
}

func RateLimitMiddleware(bot domain.BotClient, limiter *ratelimit.Limiter) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, update tgbotapi.Update) error {
			userID, chat := extractIDs(update)
			if userID == 0 || chat == 0 {
				return next(ctx, update)
			}
			if !limiter.Allow(userID) {
				_, _ = bot.Send(tgbotapi.NewMessage(chat, "⚠️ Too many messages. Please slow down."))
				return nil
			}
			return next(ctx, update)
		}
	}
}

func extractIDs(update tgbotapi.Update) (int64, int64) {
	if update.Message != nil {
		return update.Message.From.ID, update.Message.Chat.ID
	}
	if update.CallbackQuery != nil {
		return update.CallbackQuery.From.ID, update.CallbackQuery.Message.Chat.ID
	}
	return 0, 0
}

func chatID(update tgbotapi.Update) int64 {
	_, c := extractIDs(update)
	return c
}
