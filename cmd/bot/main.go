package main

import (
	"context"
	"database/sql"
	"os"
	"os/signal"
	"syscall"

	"github.com/yourusername/moneytracker/app"
	"github.com/yourusername/moneytracker/config"
	"github.com/yourusername/moneytracker/internal/logger"
	"github.com/yourusername/moneytracker/internal/ratelimit"
	llmparser "github.com/yourusername/moneytracker/parser/llm"
	ruleparser "github.com/yourusername/moneytracker/parser/rule"
	"github.com/yourusername/moneytracker/store/mysql"
	"github.com/yourusername/moneytracker/transport/telegram"
)

func main() {
	cfg, err := config.Load(".env")
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	log := logger.New(cfg.Debug)

	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		log.Error("mysql open failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Error("mysql unreachable", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := mysql.RunMigrations(ctx, db, "schema.sql"); err != nil {
		log.Error("migration failed", "error", err)
		os.Exit(1)
	}

	store := mysql.NewStore(db)
	rule := ruleparser.NewParser()

	var chainParser *llmparser.ChainParser
	if cfg.GeminiKey != "" {
		gemini, err := llmparser.NewGeminiParser(ctx, cfg.GeminiKey, cfg.GeminiModel, log)
		if err != nil {
			log.Warn("gemini disabled", "error", err)
			chainParser = llmparser.NewChainParser(rule, nil, cfg.LLMThreshold)
		} else {
			chainParser = llmparser.NewChainParser(rule, gemini, cfg.LLMThreshold)
		}
	} else {
		chainParser = llmparser.NewChainParser(rule, nil, cfg.LLMThreshold)
	}

	expenseService := app.NewExpenseService(store, chainParser, app.RealClock{})
	reportService := app.NewReportService(store, app.RealClock{})

	botClient, err := telegram.NewAPIClient(cfg.Token)
	if err != nil {
		log.Error("telegram client init failed", "error", err)
		os.Exit(1)
	}

	handler := telegram.NewHandler(botClient, store, expenseService, reportService, log)
	limiter := ratelimit.New(cfg.RateRPS, cfg.RateBurst)

	runner := telegram.NewRunner(botClient, handler, telegram.Config{
		Mode:          cfg.Mode,
		WebhookURL:    cfg.WebhookURL,
		WebhookAddr:   cfg.WebhookAddr,
		WebhookPath:   cfg.WebhookPath,
		WebhookSecret: cfg.WebhookSecret,
		Workers:       cfg.Workers,
		QueueSize:     cfg.QueueSize,
	}, log, limiter)

	log.Info("bot starting", "mode", cfg.Mode)
	if err := runner.Run(ctx); err != nil {
		log.Error("bot stopped with error", "error", err)
		os.Exit(1)
	}
	log.Info("bot stopped")
}
