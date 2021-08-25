package persistence

import "time"

type AlertPersister interface {
	GetLatestAlert() (AlertTime, bool, error)
	SaveAlert(d time.Time) error
}
