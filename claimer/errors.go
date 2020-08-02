package main

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

type ClaimError interface {
	Error() string
}

type ErrorFailed struct {
	Err error
}

func NewErrorFailed(err error) ErrorFailed {
	return ErrorFailed{
		Err: err,
	}
}

func (ef ErrorFailed) Error() string {
	return ef.Err.Error()
}

type ErrorRetryable struct {
	Err error
}

func NewErrorRetryable(err error) ErrorRetryable {
	return ErrorRetryable{
		Err: err,
	}
}

func (er ErrorRetryable) Error() string {
	return er.Err.Error()
}

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

		if fmt.Sprintf("%T", err) == "main.ErrorRetryable" {
			log.Info("retrying: ", err.Error()) // TODO: can remove check later
		}
	}
	log.Error(fmt.Errorf("timed out after %d attempts, last error: %s", attempts, err.Error()))
	return
}
