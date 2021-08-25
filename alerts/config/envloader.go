package config

import "os"

// EnvLoader loads keys from os environment
type EnvLoader struct {
}

// Verify interface compliance at compile time
var _ ConfigLoader = (*EnvLoader)(nil)

// Get retrieves key from environment
func (l *EnvLoader) Get(key string) string {
	return os.Getenv(key)
}
