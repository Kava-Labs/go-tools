package persistence

import "time"

type AlertPersister interface {
	GetLatestAlert(tableName string, serviceName string, rpcUrl string) (AlertTime, bool, error)
	SaveAlert(tableName string, serviceName string, rpcUrl string, d time.Time) error
}
