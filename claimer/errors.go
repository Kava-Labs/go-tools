package main

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// ClaimError is an interface for claimer bot errors
type ClaimError interface {
	Error() string // Error function implementation makes ClaimError type compatible with err
}

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

// Retry retries the given function for n attempts, sleeping x seconds between attempt
func Retry(attempts int, sleep time.Duration, f func() ClaimError) {
	var err error
	for i := 0; ; i++ {
		err = f()
		if err == nil {
			return
		}

		if i >= (attempts - 1) {
			break
		}

		if fmt.Sprintf("%T", err) == "main.ErrorFailed" {
			log.Error("failed: ", err.Error())
			return
		}

		time.Sleep(sleep)
		log.Info("retrying: ", err.Error())
	}
	log.Error(fmt.Errorf("timed out after %d attempts, last error: %s", attempts, err.Error()))
	return
}
