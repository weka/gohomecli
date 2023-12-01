package utils

import (
	"regexp"
	"time"
)

func FormatTime(t time.Time) string {
	return Colorize(ColorCyan, t.Format(time.RFC3339))
}

func ParseTime(text string) (time.Time, error) {
	if text == "" {
		return time.Time{}, nil
	}
	result, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return result, err
	}
	return result, nil
}

func FormatBoolean(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func FormatUUID(uuid string) string {
	return Colorize(ColorYellow, uuid)
}

var nodeIDPattern = regexp.MustCompile(`^NodeId<(\d+)>$`)

func FormatNodeID(nodeID string) string {
	submatches := nodeIDPattern.FindStringSubmatch(nodeID)
	if submatches == nil || len(submatches) != 2 {
		return nodeID
	}
	return submatches[1]
}

func FormatEventType(eventType string) string {
	return Colorize(ColorBlue, eventType)
}

func FormatEventSeverity(severity string) string {
	switch severity {
	case "DEBUG":
		return Colorize(ColorDarkGrey, severity)
	case "INFO":
		return severity
	case "WARNING":
		return Colorize(ColorYellow, severity)
	case "MINOR":
		return Colorize(ColorRed, severity)
	case "MAJOR":
		return Colorize(ColorRed, severity)
	case "CRITICAL":
		return Colorize(ColorBrightRed, severity)
	}
	return severity
}
