package container

import (
	"context"
	"github.com/Arbuz-ignor/template/internal/config"
	httpserver "github.com/Arbuz-ignor/template/internal/http"
	createhandler "github.com/Arbuz-ignor/template/internal/http/trip/create"
	finishhandler "github.com/Arbuz-ignor/template/internal/http/trip/finish"
	gethandler "github.com/Arbuz-ignor/template/internal/http/trip/get"
	"github.com/Arbuz-ignor/template/internal/infra/postgres"
	triprepo "github.com/Arbuz-ignor/template/internal/repository/trip"
	"github.com/Arbuz-ignor/template/internal/usecase/trip/create"
	"github.com/Arbuz-ignor/template/internal/usecase/trip/finish"
	"github.com/Arbuz-ignor/template/internal/usecase/trip/get"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
	"os"
)

type App struct {
	Server *http.Server
	Pool   *pgxpool.Pool
	Logger *slog.Logger
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})).With("service", "trip-service")
	pool, err := postgres.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}
	repo := triprepo.NewRepository(pool, cfg.DatabaseQueryTime)
	tx := postgres.NewTxManager(pool, cfg.DatabaseQueryTime)
	handlers := httpserver.Handlers{Create: createhandler.New(create.New(repo, tx)).Handle, Get: gethandler.New(get.New(repo)).Handle, Finish: finishhandler.New(finish.New(repo, tx)).Handle}
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: httpserver.Router(pool, handlers, cfg.DatabaseQueryTime), ReadTimeout: cfg.HTTPReadTimeout, ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout, WriteTimeout: cfg.HTTPWriteTimeout, IdleTimeout: cfg.HTTPIdleTimeout}
	return &App{Server: server, Pool: pool, Logger: logger}, nil
}
