package backup

import (
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

func normalizeCronSpec(schedule string) string {
	schedule = strings.TrimSpace(schedule)
	if schedule == "" {
		return "0 2 * * *"
	}
	parts := strings.Fields(schedule)
	if len(parts) == 5 {
		return "0 " + schedule
	}
	return schedule
}

func cronDueNow(schedule string, lastRun *time.Time, now time.Time) bool {
	spec := normalizeCronSpec(schedule)
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	sched, err := parser.Parse(spec)
	if err != nil {
		if lastRun == nil {
			return true
		}
		return now.Sub(*lastRun) >= 24*time.Hour
	}
	from := now.Add(-2 * time.Minute)
	if lastRun != nil && lastRun.After(from) {
		from = lastRun.Add(time.Second)
	}
	next := sched.Next(from)
	return !next.After(now)
}
