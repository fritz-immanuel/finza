package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/yourusername/moneytracker/internal/ratelimit"
	"github.com/yourusername/moneytracker/internal/userlock"
	"github.com/yourusername/moneytracker/internal/workerpool"
)

type APIClient struct {
	api *tgbotapi.BotAPI
}

func NewAPIClient(token string) (*APIClient, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	return &APIClient{api: api}, nil
}

func (c *APIClient) Send(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
	return c.api.Send(msg)
}

func (c *APIClient) Request(m tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	return c.api.Request(m)
}

func (c *APIClient) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return c.api.GetUpdatesChan(config)
}

func (c *APIClient) SetWebhook(config tgbotapi.WebhookConfig) error {
	_, err := c.api.Request(config)
	return err
}

type Config struct {
	Mode          string
	WebhookURL    string
	WebhookAddr   string
	WebhookPath   string
	WebhookSecret string
	Workers       int
	QueueSize     int
}

type Runner struct {
	bot     *APIClient
	handler *Handler
	cfg     Config
	logger  *slog.Logger
	lock    *userlock.Locker
	limiter *ratelimit.Limiter
}

func NewRunner(bot *APIClient, handler *Handler, cfg Config, logger *slog.Logger, limiter *ratelimit.Limiter) *Runner {
	if cfg.Workers < 1 {
		cfg.Workers = 4
	}
	if cfg.QueueSize < 1 {
		cfg.QueueSize = 100
	}
	return &Runner{bot: bot, handler: handler, cfg: cfg, logger: logger, lock: userlock.New(), limiter: limiter}
}

func (r *Runner) Run(ctx context.Context) error {
	wrapped := Chain(
		r.handler.HandleUpdate,
		RecoveryMiddleware(r.logger, r.bot),
		RateLimitMiddleware(r.bot, r.limiter),
		LoggerMiddleware(r.logger, "update"),
	)

	pool := workerpool.New[tgbotapi.Update](r.cfg.Workers, r.cfg.QueueSize, func(ctx context.Context, update tgbotapi.Update) {
		userID, _ := extractIDs(update)
		unlock := r.lock.Lock(userID)
		defer unlock()

		if err := wrapped(ctx, update); err != nil {
			_, chat := extractIDs(update)
			if chat != 0 {
				r.handler.sendError(chat, err)
			}
		}
	})
	defer pool.Close()

	go r.cleanupLimiter(ctx)

	switch r.cfg.Mode {
	case "webhook":
		return r.runWebhook(ctx, pool)
	default:
		return r.runPolling(ctx, pool)
	}
}

func (r *Runner) cleanupLimiter(ctx context.Context) {
	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.limiter.Cleanup()
		}
	}
}

func (r *Runner) runPolling(ctx context.Context, pool *workerpool.Pool[tgbotapi.Update]) error {
	updates := r.bot.GetUpdatesChan(tgbotapi.NewUpdate(0))
	for {
		select {
		case <-ctx.Done():
			return nil
		case upd := <-updates:
			if ok := pool.Submit(upd); !ok {
				r.logger.Warn("worker queue full; dropping update", "update_id", upd.UpdateID)
			}
		}
	}
}

func (r *Runner) runWebhook(ctx context.Context, pool *workerpool.Pool[tgbotapi.Update]) error {
	webhook, err := tgbotapi.NewWebhook(r.cfg.WebhookURL + r.cfg.WebhookPath)
	if err != nil {
		return fmt.Errorf("new webhook: %w", err)
	}
	if err := r.bot.SetWebhook(webhook); err != nil {
		return fmt.Errorf("set webhook: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(r.cfg.WebhookPath, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.cfg.WebhookSecret != "" {
			if req.Header.Get("X-Telegram-Bot-Api-Secret-Token") != r.cfg.WebhookSecret {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
		var update tgbotapi.Update
		if err := json.NewDecoder(req.Body).Decode(&update); err != nil {
			r.logger.Error("handle webhook update", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = pool.Submit(update)
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{Addr: r.cfg.WebhookAddr, Handler: mux}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("webhook server: %w", err)
	}
	return nil
}
