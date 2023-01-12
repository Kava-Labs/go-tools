package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"github.com/kava-labs/go-tools/auction-audit/config"
)

func GetFileName(prefix string, config config.Config) string {
	return fmt.Sprintf(
		"%s_%d_%d.csv",
		prefix, config.StartHeight, config.EndHeight,
	)
}

func GetFileOutput(prefix string, config config.Config) (io.Writer, error) {
	fileName := GetFileName(prefix, config)
	return os.Create(fileName)
}

func WriteCsvRecords(
	destination io.Writer,
	headers []string,
	records [][]string,
) error {
	w := csv.NewWriter(destination)
	if err := w.Write(headers); err != nil {
		return err
	}

	// Flushes internally
	if err := w.WriteAll(records); err != nil {
		return err
	}

	return w.Error()
}

func WriteCsv(
	destination io.Writer,
	headers []string,
	input CsvWritable,
) error {
	return WriteCsvRecords(destination, headers, input.ToRecords())
}

type CsvWritable interface {
	ToRecords() [][]string
}
