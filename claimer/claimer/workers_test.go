package claimer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPollWithBackoff(t *testing.T) {

	pollTimes := []time.Duration{}
	start := time.Now()

	err := pollWithBackoff(time.Second, 100*time.Millisecond, func() (bool, error) {
		pollTimes = append(pollTimes, time.Since(start))
		return false, nil
	})
	require.Error(t, err)

	expectedPollTimes := []time.Duration{
		0 * time.Millisecond,
		100 * time.Millisecond,
		300 * time.Millisecond,
		700 * time.Millisecond,
	}

	// check actual times are within some percent of expected
	// drop first 0 time as the error is variable
	require.InEpsilonSlice(t, expectedPollTimes[1:], pollTimes[1:], 0.05)
}
