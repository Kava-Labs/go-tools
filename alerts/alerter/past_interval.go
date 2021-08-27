package alerter

import (
	"fmt"
	"time"

	"github.com/kava-labs/go-tools/alerts/persistence"
)

// PastInterval returns whether or not it has passed the internal frequency
func IsPastInterval(db persistence.AlertPersister, duration time.Duration) (bool, persistence.AlertTime, error) {
	lastAlert, found, err := db.GetLatestAlert()
	if err != nil {
		return false, persistence.AlertTime{}, fmt.Errorf("Failed to fetch latest alert time: %v", err.Error())
	}

	// If current time in UTC is before (previous timestamp + alert frequency), skip alert
	if found && time.Now().UTC().Before(lastAlert.Timestamp.Add(duration)) {
		return false, lastAlert, nil
	}

	if err := db.SaveAlert(time.Now().UTC()); err != nil {
		return false, persistence.AlertTime{}, fmt.Errorf("Failed to save alert time to DynamoDb: %v", err.Error())
	}

	return true, lastAlert, nil
}
