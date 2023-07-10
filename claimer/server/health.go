package server

import (
	"context"
	"net/http"
	"time"

	"github.com/alexliesenfeld/health"
	"github.com/kava-labs/go-tools/claimer/claimer"
	"github.com/sirupsen/logrus"
)

func healthCheckHandler(
	ctx context.Context,
	logger *logrus.Logger,
	dispatcher *claimer.Dispatcher,
) http.HandlerFunc {
	// Create a new Checker.
	checker := health.NewChecker(
		health.WithCacheDuration(1*time.Second),
		health.WithTimeout(10*time.Second),
		// Run every minute with initial delay of 3 seconds. Not run each HTTP request
		health.WithPeriodicCheck(60*time.Second, 3*time.Second, health.Check{
			Name: "kava grpc",
			Check: func(ctx context.Context) error {
				_, err := dispatcher.KavaClient.GetChainID()
				return err
			},
		}),
		health.WithPeriodicCheck(60*time.Second, 3*time.Second, health.Check{
			Name: "bnb http",
			Check: func(ctx context.Context) error {
				_, err := dispatcher.BnbClient.Status()
				return err
			},
		}),
		// Runs when health status changes
		health.WithStatusListener(func(ctx context.Context, state health.CheckerState) {
			logger.
				WithField("status", string(state.Status)).
				Debugf("health status changed")
		}),
	)

	return health.NewHandler(checker)
}
