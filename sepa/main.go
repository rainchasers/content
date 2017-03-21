package main

import (
	"github.com/rainchasers/report"
	"os"
	"strconv"
	"time"
)

const (
	maxDownloadPerSecond = 1
	maxPublishPerSecond  = 20
	httpTimeoutInSeconds = 60
	httpUserAgent        = "Rainchaser Bot <hello@rainchasers.com>"
)

// Responds to environment variables:
//   UPDATE_EVERY_X_SECONDS (default 15*60)
//   SHUTDOWN_AFTER_X_SECONDS (default 7*24*60*60)
//   PROJECT_ID (no default, blank for validation mode)
//   LATEST_PUBSUB_TOPIC (no default, blank for validation mode)
//   HISTORY_PUBSUB_TOPIC (no default, blank for validation mode)
//
func main() {
	if err := run(); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	// setup telemetry and logging
	report.StdOut()
	report.Global(report.Data{"service": "sepa", "daemon": time.Now().Format("v2006-01-02-15-04-05")})
	report.RuntimeStatsEvery(30 * time.Second)
	defer report.Drain()

	// parse env vars
	updatePeriodSeconds, err := strconv.Atoi(os.Getenv("UPDATE_EVERY_X_SECONDS"))
	if err != nil {
		updatePeriodSeconds = 15 * 60
	}
	shutdownDeadline, err := strconv.Atoi(os.Getenv("SHUTDOWN_AFTER_X_SECONDS"))
	if err != nil {
		shutdownDeadline = 7 * 24 * 60 * 60
	}
	shutdownC := time.NewTimer(time.Second * time.Duration(shutdownDeadline)).C
	projectId := os.Getenv("PROJECT_ID")
	latestTopicName := os.Getenv("LATEST_PUBSUB_TOPIC")
	historyTopicName := os.Getenv("HISTORY_PUBSUB_TOPIC")

	// decision on whether validating logs
	isValidating := projectId == ""
	var logs *LogBuffer
	if isValidating {
		logs = trackLogs()
	}
	report.Info("daemon.start", report.Data{
		"update_period":        updatePeriodSeconds,
		"shutdown_deadline":    shutdownDeadline,
		"project_id":           projectId,
		"latest_pubsub_topic":  latestTopicName,
		"history_pubsub_topic": historyTopicName,
	})

	// discover SEPA gauging stations
	refSnapshots, err := discover()
	if err != nil {
		report.Action("discovered.failed", report.Data{"error": err.Error()})
		return err
	}
	if isValidating {
		refSnapshots = refSnapshots[0:5]
	}
	report.Info("discovered.ok", report.Data{"count": len(refSnapshots)})

	// calculate tick rate and spawn individual gauge download CSVs
	tickerMs := updatePeriodSeconds * 1000 / len(refSnapshots)
	minTickerMs := 1000 / maxDownloadPerSecond
	if tickerMs < minTickerMs {
		tickerMs = minTickerMs
	}
	n := 0
	ticker := time.NewTicker(time.Millisecond * time.Duration(tickerMs))

updateLoop:
	for {
		i := n % len(refSnapshots)

		tick := report.Tick()
		readings, err := getReadings(refSnapshots[i])
		if err != nil {
			report.Tock(tick, "updated.failed", report.Data{
				"url":   refSnapshots[i].DataURL,
				"error": err.Error(),
			})
		} else {
			report.Tock(tick, "updated.ok", report.Data{
				"url":   refSnapshots[i].DataURL,
				"count": len(readings),
			})
		}

		n = n + 1
		select {
		case <-ticker.C:
		case <-shutdownC:
			break updateLoop
		}
	}
	ticker.Stop()

	// validate log stream on shutdown if required
	if isValidating {
		report.Drain()
		expect := map[string]int{
			"discovered.ok": VALIDATE_IS_PRESENT,
			"updated.ok":    VALIDATE_IS_PRESENT,
		}
		err := validateLogStream(logs, expect)
		if err != nil {
			return err
		}
	}

	return nil
}
