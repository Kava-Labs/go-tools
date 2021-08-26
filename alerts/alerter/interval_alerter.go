package alerter

import (
	"fmt"
	"time"

	"github.com/kava-labs/go-tools/alerts/persistence"
)

// FrequencyLimiter wraps holds a AlertPersister to provide a way to easily
// limit function calls in a given frequency
type FrequencyLimiter struct {
	db        persistence.AlertPersister
	frequency time.Duration
}

func NewFrequencyLimiter(db persistence.AlertPersister, frequency time.Duration) FrequencyLimiter {
	return FrequencyLimiter{
		db,
		frequency,
	}
}

func (a *FrequencyLimiter) Exec(fPass func() error, fFail func(lastAlert persistence.AlertTime) error) error {
	lastAlert, found, err := a.db.GetLatestAlert()
	if err != nil {
		return fmt.Errorf("Failed to fetch latest alert time: %v", err.Error())
	}

	// If current time in UTC is before (previous timestamp + alert frequency), skip alert
	if found && time.Now().UTC().Before(lastAlert.Timestamp.Add(a.frequency)) {
		return fFail(lastAlert)
	}

	if err := fPass(); err != nil {
		return err
	}

	if err := a.db.SaveAlert(time.Now().UTC()); err != nil {
		return fmt.Errorf("Failed to save alert time to DynamoDb: %v", err.Error())
	}

	return nil
}
