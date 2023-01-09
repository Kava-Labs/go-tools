package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

func GetFileName(prefix string, config Config) string {
	return fmt.Sprintf(
		"%s_%d_%d.csv",
		prefix, config.StartHeight, config.EndHeight,
	)
}

func GetFileOutput(prefix string, config Config) (io.Writer, error) {
	fileName := GetFileName(prefix, config)
	return os.Create(fileName)
}

func WriteCsv(
	destination io.Writer,
	headers []string,
	records [][]string,
) error {
	w := csv.NewWriter(destination)
	// Flushes internally
	return w.WriteAll(records)
}
