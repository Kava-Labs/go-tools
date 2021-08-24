package config

import "os"

// EnvLoader loads keys from os environment
type EnvLoader struct {
}

// Get retrieves key from environment
func (l *EnvLoader) Get(key string) string {
	return os.Getenv(key)
}
