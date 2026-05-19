// Command coralogix-metrics-usage scans Coralogix dashboards, alerts, and SLOs for PromQL
// metric references and compares them to the live metrics catalog.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/avi/coralogix-metrics-usage/internal/coralogix"
	"github.com/avi/coralogix-metrics-usage/internal/metricusage"
	"github.com/avi/coralogix-metrics-usage/internal/region"
	"github.com/avi/coralogix-metrics-usage/internal/scan"
)

func main() {
	os.Exit(run())
}

func run() int {
	regionFlag := flag.String("region", "", "Coralogix region or domain (eu1, eu2.coralogix.com, api.eu2.coralogix.com, …)")
	keyFlag := flag.String("key", "", "Coralogix API key (Bearer)")
	outputDir := flag.String("output-dir", ".", "directory for report outputs (JSON, CSV per-series + per-metric rollup, OTEL YAML)")
	lookbackHours := flag.Float64("series-lookback-hours", 25, "time window for Prometheus series discovery")
	seriesLimit := flag.Int("series-limit-per-metric", 50_000, "max series rows per metric name")
	workers := flag.Int("workers", 8, "parallel series fetches")
	timeoutSec := flag.Int("timeout-sec", 120, "HTTP client timeout per request")
	usageDays := flag.Int("usage-lookback-days", 7, "rolling inclusive UTC calendar days ending today for CX unit_usage (0 skips rolling window; use --usage-billing-calendar-months instead)")
	usageMonths := flag.Int("usage-billing-calendar-months", 0, "if >0, CX unit_usage window is the last N complete UTC calendar months (overrides rolling days when both set); 0 uses rolling days only")
	skipBilling := flag.Bool("skip-billing", false, "skip Metrics Usage API (no unit_usage on output)")
	skipDashboards := flag.Bool("skip-dashboards", false, "skip dashboard catalog and definitions (omit Dashboard preset)")
	skipAlerts := flag.Bool("skip-alerts", false, "skip alert definitions v3 (omit Alerts preset)")
	skipSLO := flag.Bool("skip-slo", false, "skip SLO list (omit SLO preset)")
	quiet := flag.Bool("quiet", false, "disable progress status line on stderr")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Required flags: --region and --key\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *usageDays < 0 || *usageMonths < 0 {
		fmt.Fprintln(os.Stderr, "usage lookback days and billing calendar months must be non-negative")
		return 2
	}

	if *regionFlag == "" || *keyFlag == "" {
		flag.Usage()
		return 2
	}

	apiHost, err := region.ResolveAPIHost(*regionFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	client := coralogix.NewClient(apiHost, *keyFlag, time.Duration(*timeoutSec)*time.Second)
	ctx := context.Background()

	var billingClient *metricusage.Client
	if !*skipBilling && (*usageDays > 0 || *usageMonths > 0) {
		billingClient, err = metricusage.NewClient(apiHost, *keyFlag)
		if err != nil {
			fmt.Fprintln(os.Stderr, "billing client:", err)
			return 1
		}
		defer billingClient.Close()
	}

	rep, err := scan.Run(ctx, client, scan.Options{
		SeriesLookback:             time.Duration(*lookbackHours * float64(time.Hour)),
		SeriesLimitPerMetric:       *seriesLimit,
		Workers:                    *workers,
		UsageLookbackDays:          *usageDays,
		UsageBillingCalendarMonths: *usageMonths,
		Billing:                    billingClient,
		Quiet:                      *quiet,
		SkipDashboards:             *skipDashboards,
		SkipAlerts:                 *skipAlerts,
		SkipSLOs:                   *skipSLO,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan failed:", err)
		return 1
	}

	if err := rep.Write(*outputDir); err != nil {
		fmt.Fprintln(os.Stderr, "write output:", err)
		return 1
	}

	fmt.Printf("Wrote %s/metric_usage_summary.json\n", *outputDir)
	fmt.Printf("Wrote %s/metric_usage_unused_series.json\n", *outputDir)
	fmt.Printf("Wrote %s/metric_usage_unused_by_cost.json\n", *outputDir)
	fmt.Printf("Wrote %s/metric_usage_unused_by_cost.csv\n", *outputDir)
	fmt.Printf("Wrote %s/metric_usage_unused_by_metric.json\n", *outputDir)
	fmt.Printf("Wrote %s/metric_usage_unused_by_metric.csv\n", *outputDir)
	fmt.Printf("Wrote %s/metric_usage_otel_processors.yaml\n", *outputDir)
	m := rep.Meta
	fmt.Printf(
		"Scanned dashboards=%d alerts=%d slos=%d; catalog series=%d; used=%d; unused=%d; "+
			"referenced metric absent in lookback=%d; no label match=%d; no metric name=%d; "+
			"series with billing=%d; unused with billing=%d; coralogix cx_* metric names skipped=%d\n",
		m.DashboardsScanned, m.AlertsScanned, m.SLOsScanned,
		m.DistinctSeriesInCatalog,
		len(rep.UsedSeriesInCatalog),
		len(rep.UnusedSeriesInCatalog),
		len(rep.ReferencedSelectorsMetricAbsentInTimeseriesWindow),
		len(rep.ReferencedSelectorsMetricPresentButNoSeriesMatches),
		len(rep.ReferencedSelectorsWithoutMetricName),
		m.SeriesWithBillingData,
		m.UnusedSeriesWithBilling,
		m.CoralogixInternalMetricNamesSkipped,
	)
	if m.MetricsTruncatedAtSeriesLimit > 0 {
		fmt.Fprintf(os.Stderr,
			"Warning: %d metric(s) hit the per-metric series limit; unused list may be incomplete.\n",
			m.MetricsTruncatedAtSeriesLimit,
		)
	}
	if *skipDashboards || *skipAlerts || *skipSLO {
		var skipped []string
		if *skipDashboards {
			skipped = append(skipped, "dashboards")
		}
		if *skipAlerts {
			skipped = append(skipped, "alerts")
		}
		if *skipSLO {
			skipped = append(skipped, "SLOs")
		}
		fmt.Fprintf(os.Stderr,
			"Note: correlation skipped for %s — used/unused and OTEL output reflect PromQL only from scanned sources (see report warnings).\n",
			strings.Join(skipped, ", "))
	}
	return 0
}
