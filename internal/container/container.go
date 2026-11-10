package container

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"

	"github.com/Arbuz-ignor/template/internal/config"
	httpserver "github.com/Arbuz-ignor/template/internal/http"
	"github.com/Arbuz-ignor/template/internal/http/middleware"
	createhandler "github.com/Arbuz-ignor/template/internal/http/trip/create"
	finishhandler "github.com/Arbuz-ignor/template/internal/http/trip/finish"
	gethandler "github.com/Arbuz-ignor/template/internal/http/trip/get"
	"github.com/Arbuz-ignor/template/internal/infra/postgres"
	"github.com/Arbuz-ignor/template/internal/infra/telemetry"
	triprepo "github.com/Arbuz-ignor/template/internal/repository/trip"
	"github.com/Arbuz-ignor/template/internal/usecase/trip/create"
	"github.com/Arbuz-ignor/template/internal/usecase/trip/finish"
	"github.com/Arbuz-ignor/template/internal/usecase/trip/get"
)

type App struct {
	Server    *http.Server
	Pool      *pgxpool.Pool
	Logger    *slog.Logger
	Telemetry *telemetry.Telemetry
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	signals, err := telemetry.New(ctx, cfg.LogLevel)
	if err != nil {
		return nil, err
	}
	logger := signals.Logger
	dbTracer, err := telemetry.NewDatabaseTracer(signals.Traces, signals.Meters)
	if err != nil {
		return nil, err
	}
	pool, err := postgres.Open(ctx, cfg, dbTracer)
	if err != nil {
		_ = signals.Shutdown(ctx)
		return nil, err
	}
	business, err := telemetry.NewBusiness(signals.Meters.Meter("trip-service/business"))
	if err != nil {
		pool.Close()
		_ = signals.Shutdown(ctx)
		return nil, err
	}
	middlewares, err := middleware.New(logger, signals.Traces, signals.Meters)
	if err != nil {
		pool.Close()
		_ = signals.Shutdown(ctx)
		return nil, err
	}
	repo := triprepo.NewRepository(pool, cfg.DatabaseQueryTime)
	_, err = signals.Meters.Meter("trip-service/business").Int64ObservableGauge("tripgo.trips.active", metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
		count, err := repo.CountActive(ctx)
		if err == nil {
			observer.Observe(count)
		}
		return err
	}))
	if err != nil {
		pool.Close()
		_ = signals.Shutdown(ctx)
		return nil, err
	}
	tx := postgres.NewTxManager(pool, cfg.DatabaseQueryTime)
	handlers := httpserver.Handlers{Create: createhandler.New(create.New(repo, tx, logger, business)).Handle, Get: gethandler.New(get.New(repo)).Handle, Finish: finishhandler.New(finish.New(repo, tx, logger, business)).Handle}
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: httpserver.Router(pool, handlers, cfg.DatabaseQueryTime, middlewares.Recover, middlewares.Tracing, middlewares.Logging, middlewares.Metrics), ReadTimeout: cfg.HTTPReadTimeout, ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout, WriteTimeout: cfg.HTTPWriteTimeout, IdleTimeout: cfg.HTTPIdleTimeout}
	return &App{Server: server, Pool: pool, Logger: logger, Telemetry: signals}, nil
}
