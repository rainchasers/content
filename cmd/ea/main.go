package main

import (
	"context"
	"os"
	"time"

	"github.com/robtuley/rainchasers/internal/daemon"
	"github.com/robtuley/rainchasers/internal/ea"
	"github.com/robtuley/rainchasers/internal/gauge"
	"github.com/robtuley/rainchasers/internal/queue"
)

// Responds to environment variables:
//   PROJECT_ID (no default, blank skips publish)
//   PUBSUB_TOPIC (no default)
func main() {
	cfg := config{
		ProjectID:                os.Getenv("PROJECT_ID"),
		TopicName:                os.Getenv("PUBSUB_TOPIC"),
		RefreshPeriodInSeconds:   15 * 60,
		MaxPublishPerSecond:      30,
		ExitAfterXConsecutiveErr: 3,
	}

	d := daemon.New("ea")
	d.Run(context.Background(), cfg.run)
	d.CloseAfter(4 * time.Hour)
	d.Wait()
	if err := d.Err(); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}

type config struct {
	ProjectID                string
	TopicName                string
	RefreshPeriodInSeconds   int
	MaxPublishPerSecond      int
	ExitAfterXConsecutiveErr int
}

func (cfg config) run(ctx context.Context, d *daemon.Supervisor) error {
	// discover EA gauging stations
	stations, span := ea.Discover(ctx)
	d.Trace(span)
	if err := span.Err(); err != nil {
		return err
	}

	nConsecutiveErr := 0
updateLoop:
	for {
		err := func(ctx context.Context) error {
			// get all recent readings
			readings, rSpan := ea.Recent(ctx)
			if err := rSpan.Err(); err != nil {
				d.Trace(rSpan)
				return err
			}

			// open connection to pubsub
			topic, tSpan := queue.New(ctx, cfg.ProjectID, cfg.TopicName)
			d.Trace(rSpan.FollowedBy(tSpan))
			if err := tSpan.Err(); err != nil {
				return err
			}
			defer topic.Stop()

			// ticker to spread readings publish over the full refresh period
			every := cfg.durationBetweenPublish(len(readings))
			ticker := time.NewTicker(every)
			defer ticker.Stop()

			// publish readings
			for id, r := range readings {
				s, ok := stations[id]
				if !ok {
					continue
				}

				span := topic.Publish(ctx, &gauge.Snapshot{
					Station:  s,
					Readings: []gauge.Reading{r},
				})
				d.Trace(span)
				if err := span.Err(); err != nil {
					return err
				}

				select {
				case <-ticker.C:
				case <-ctx.Done():
					// exit early on shutdown
					return nil
				}
			}

			return nil
		}(ctx)

		if err != nil {
			nConsecutiveErr++
			if nConsecutiveErr >= cfg.ExitAfterXConsecutiveErr {
				// ignore a few isolated errors, but if
				// many consecutive bubble up to restart
				return err
			}
		} else {
			nConsecutiveErr = 0
		}

		// break loop on shutdown signal
		select {
		case <-ctx.Done():
			break updateLoop
		default:
		}
	}

	return nil
}

func (cfg config) durationBetweenPublish(total int) time.Duration {
	ms := cfg.RefreshPeriodInSeconds * 1000 / total
	min := 1
	if cfg.MaxPublishPerSecond > 0 {
		min = 1000 / cfg.MaxPublishPerSecond
	}
	if ms < min {
		ms = min
	}
	return time.Millisecond * time.Duration(ms)
}
