package health

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/alexliesenfeld/health"
	"github.com/go-chi/chi/v5"
	"github.com/kava-labs/go-tools/deputy-claimer/claim"
	"github.com/rs/zerolog"
)

const HEALTH_CHECK_LISTEN_ADDR_KEY = "HEALTH_CHECK_LISTEN_ADDR"

func StartHealthCheckService(
	ctx context.Context,
	logger zerolog.Logger,
	kavaClaimer claim.KavaClaimer,
) {
	// Create a new Checker.
	checker := health.NewChecker(
		health.WithCacheDuration(1*time.Second),
		health.WithTimeout(10*time.Second),
		// Run every minute with initial delay of 3 seconds. Not run each HTTP request
		health.WithPeriodicCheck(60*time.Second, 3*time.Second, health.Check{
			Name: "kava grpc",
			Check: func(ctx context.Context) error {
				_, err := kavaClaimer.KavaClient.GetChainID()
				logger.Debug().Err(err).Msg("kava grpc periodic health check")
				return err
			},
		}),
		health.WithPeriodicCheck(60*time.Second, 3*time.Second, health.Check{
			Name: "bnb grpc",
			Check: func(ctx context.Context) error {
				_, err := kavaClaimer.BnbClient.Status()
				logger.Debug().Err(err).Msg("bnb periodic health check")
				return err
			},
		}),
		// Runs when health status changes
		health.WithStatusListener(func(ctx context.Context, state health.CheckerState) {
			logger.
				Debug().
				Interface("state", state).
				Msg("health status changed")
		}),
	)

	r := chi.NewRouter()
	r.Get("/health", health.NewHandler(checker))

	addr := os.Getenv(HEALTH_CHECK_LISTEN_ADDR_KEY)
	if addr == "" {
		addr = ":8080"
	}

	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		logger.
			Info().
			Msgf("healthcheck server listening on %s", server.Addr)

		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("failed to start healthcheck server")
		}
	}()
}
