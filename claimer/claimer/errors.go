package claimer

import (
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// ErrorFailed is the type of error thrown when a claim request is non-retryable
type ErrorFailed struct {
	Err error
}

// NewErrorFailed instantiates a new instance of ErrorFailed
func NewErrorFailed(err error) ErrorFailed {
	return ErrorFailed{
		Err: err,
	}
}

// Error implementation required for interface compliance
func (ef ErrorFailed) Error() string {
	return ef.Err.Error()
}

// ErrorRetryable is the type of error thrown when a claim request is retryable
type ErrorRetryable struct {
	Err error
}

// NewErrorRetryable instantiates a new instance of ErrorRetryable
func NewErrorRetryable(err error) ErrorRetryable {
	return ErrorRetryable{
		Err: err,
	}
}

// Error implementation required for interface compliance
func (er ErrorRetryable) Error() string {
	return er.Err.Error()
}

// Retry retries the given function for n attempts, sleeping x seconds between attempts.
func Retry(
	attempts int,
	sleep time.Duration,
	logger log.FieldLogger,
	failureChan chan<- error,
	f func() (string, string, error),
) {
	for i := 0; ; i++ {
		txHash, recipient, err := f()
		if err == nil {
			logger.WithFields(log.Fields{
				"recipient": recipient,
				"tx_hash":   txHash,
			}).Info("claim confirmed")

			// Clear the failure channel
			failureChan <- nil
			return
		}

		if i >= (attempts - 1) {
			logger.WithFields(log.Fields{
				"recipient": recipient,
				"error":     fmt.Errorf("timed out after %d attempts, last error: %s", attempts, err.Error()),
			}).Error("claim abandoned")
			return
		}

		if errors.As(err, &ErrorFailed{}) {
			failureChan <- err

			logger.WithFields(log.Fields{
				"recipient": recipient,
				"error":     err.Error(),
			}).Error("claim failed")

			return
		}

		time.Sleep(sleep)
		logger.WithFields(log.Fields{
			"recipient": recipient,
			"error":     err.Error(),
		}).Debug("claim retrying")
	}
}
