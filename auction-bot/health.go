package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/alexliesenfeld/health"
	"github.com/go-chi/chi/v5"
	"github.com/kava-labs/go-tools/signing"
	"github.com/rs/zerolog"
)

func startHealthCheckService(
	ctx context.Context,
	logger zerolog.Logger,
	config Config,
	client GrpcClient,
	signer *signing.Signer,
) {
	// Create a new Checker.
	checker := health.NewChecker(
		health.WithCacheDuration(1*time.Second),
		health.WithTimeout(10*time.Second),
		// Run every minute with initial delay of 3 seconds. Not run each HTTP request
		health.WithPeriodicCheck(60*time.Second, 3*time.Second, health.Check{
			Name: "kava grpc",
			Check: func(ctx context.Context) error {
				_, err := client.LatestHeight()
				return err
			},
		}),
		health.WithCheck(health.Check{
			Name: "signing account",
			Check: func(ctx context.Context) error {
				return signer.GetAccountError()
			},
		}),
		// Runs when health status changes
		health.WithStatusListener(func(ctx context.Context, state health.CheckerState) {
			logger.
				Debug().
				Str("status", string(state.Status)).
				Msg("health status changed")
		}),
	)

	r := chi.NewRouter()
	r.Get("/health", health.NewHandler(checker))

	server := &http.Server{
		Addr:    config.HeathCheckListenAddr,
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
