package alerter

import (
	"fmt"
	"time"

	"github.com/kava-labs/go-tools/alerts/persistence"
)

// PastInterval returns the last alert, whether or not it can alert again, and then saves the current time as the last alert
func GetAndSaveLastAlert(db persistence.AlertPersister, duration time.Duration) (persistence.AlertTime, bool, error) {
	lastAlert, found, err := db.GetLatestAlert()
	if err != nil {
		return persistence.AlertTime{}, false, fmt.Errorf("Failed to fetch latest alert time: %v", err.Error())
	}

	// If current time in UTC is before (previous timestamp + alert frequency), skip alert
	if found && time.Now().UTC().Before(lastAlert.Timestamp.Add(duration)) {
		return lastAlert, false, nil
	}

	if err := db.SaveAlert(time.Now().UTC()); err != nil {
		return persistence.AlertTime{}, false, fmt.Errorf("Failed to save alert time to DynamoDb: %v", err.Error())
	}

	return lastAlert, true, nil
}
