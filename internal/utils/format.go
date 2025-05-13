package utils

import (
	"regexp"
	"time"
)

// nodeIDPattern is a regex pattern to extract the node ID from a node ID string
var nodeIDPattern = regexp.MustCompile(`^NodeId<(\d+)>$`)

// FormatTime formats a time
func FormatTime(t time.Time) string {
	return Colorize(colorCyan, t.Format(time.RFC3339))
}

// ParseTime parses a time string
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

// FormatBoolean formats a boolean
func FormatBoolean(b bool) string {
	if b {
		return "Yes"
	}

	return "No"
}

// FormatUUID formats a UUID
func FormatUUID(uuid string) string {
	return Colorize(colorYellow, uuid)
}

// FormatNodeID formats a node ID
func FormatNodeID(nodeID string) string {
	submatches := nodeIDPattern.FindStringSubmatch(nodeID)
	if submatches == nil || len(submatches) != 2 {
		return nodeID
	}

	return submatches[1]
}

// FormatEventType formats an event type
func FormatEventType(eventType string) string {
	return Colorize(colorBlue, eventType)
}

// FormatEventSeverity formats an event severity
func FormatEventSeverity(severity string) string {
	switch severity {
	case "DEBUG":
		return Colorize(colorDarkGrey, severity)
	case "INFO":
		return severity
	case "WARNING":
		return Colorize(colorYellow, severity)
	case "MINOR":
		return Colorize(colorRed, severity)
	case "MAJOR":
		return Colorize(colorRed, severity)
	case "CRITICAL":
		return Colorize(colorBrightRed, severity)
	}

	return severity
}
