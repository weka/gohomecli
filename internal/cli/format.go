package cli

import (
	"github.com/weka/gohomecli/internal/utils"
	"regexp"
	"time"
)

func FormatTime(t time.Time) string {
	return utils.Colorize(utils.ColorCyan, t.Format(time.RFC3339))
}

func FormatBoolean(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

var nodeIDPattern, _ = regexp.Compile("^NodeId<(\\d+)>$")

func FormatNodeID(nodeID string) string {
	submatches := nodeIDPattern.FindStringSubmatch(nodeID)
	if submatches == nil || len(submatches) != 2 {
		return nodeID
	}
	return submatches[1]
}

func FormatEventType(eventType string) string {
	return utils.Colorize(utils.ColorBlue, eventType)
}

func FormatEventSeverity(severity string) string {
	switch severity {
	case "DEBUG":
		return utils.Colorize(utils.ColorDarkGrey, severity)
	case "INFO":
		return severity
	case "WARNING":
		return utils.Colorize(utils.ColorYellow, severity)
	case "MINOR":
		return utils.Colorize(utils.ColorRed, severity)
	case "MAJOR":
		return utils.Colorize(utils.ColorRed, severity)
	case "CRITICAL":
		return utils.Colorize(utils.ColorBrightRed, severity)
	}
	return severity
}
