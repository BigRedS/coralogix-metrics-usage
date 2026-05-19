package report

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// WriteUnusedByCostCSV writes flat billing columns for spreadsheets.
func WriteUnusedByCostCSV(path string, rows []UnusedByCostRow) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	header := []string{
		"unit_usage",
		"billing_present",
		"billing_split_n",
		"metric_name",
		"series",
		"bytes_volume",
		"sample_count",
		"cardinality",
		"days_in_range",
		"labels_json",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, row := range rows {
		labelsJSON, err := json.Marshal(row.Labels)
		if err != nil {
			return err
		}
		rec := []string{
			strconv.FormatFloat(row.UnitUsage, 'g', -1, 64),
			strconv.FormatBool(row.BillingPresent),
			strconv.Itoa(row.BillingSplitN),
			row.MetricName,
			row.Series,
			strconv.FormatUint(row.BytesVolume, 10),
			strconv.FormatUint(row.SampleCount, 10),
			strconv.FormatUint(row.Cardinality, 10),
			strconv.Itoa(row.DaysInRange),
			string(labelsJSON),
		}
		if err := w.Write(rec); err != nil {
			return fmt.Errorf("write csv row for %s: %w", row.Series, err)
		}
	}
	w.Flush()
	return w.Error()
}
