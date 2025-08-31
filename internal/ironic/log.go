package ironic

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
)

var ironicLogPattern = regexp.MustCompile(
	`^(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\.\d+)\s+(\d+)\s+(\w+)\s+([^\s]+)\s+\[([^\]]*)\]\s+(.*)$`,
)

type logWriter struct {
	logger *slog.Logger
	prefix string
}

func (lw *logWriter) Write(p []byte) (n int, err error) {
	lines := strings.SplitSeq(strings.TrimSpace(string(p)), "\n")
	for line := range lines {
		if line != "" {
			lw.parseAndLog(line)
		}
	}
	return len(p), nil
}

// parseAndLog parses Ironic log lines into structured fields.
func (lw *logWriter) parseAndLog(line string) {
	matches := ironicLogPattern.FindStringSubmatch(line)
	if len(matches) == 7 {
		timestamp := matches[1]
		processID := matches[2]
		level := strings.ToUpper(matches[3])
		module := matches[4]
		requestID := matches[5]
		message := matches[6]

		// Parse message to extract function and file info if present
		msgParts := strings.Fields(message)
		var logMessage, function, fileLocation string

		if len(msgParts) > 2 {
			// Look for file path pattern at the end
			lastPart := msgParts[len(msgParts)-1]
			if strings.Contains(lastPart, "/") && strings.Contains(lastPart, ":") {
				fileLocation = lastPart
				if len(msgParts) > 1 {
					function = msgParts[len(msgParts)-2]
					logMessage = strings.Join(msgParts[:len(msgParts)-2], " ")
				}
			} else {
				logMessage = message
			}
		} else {
			logMessage = message
		}

		// Log with appropriate level and structured fields
		attrs := []slog.Attr{
			slog.String("timestamp", timestamp),
			slog.String("process_id", processID),
			slog.String("module", module),
			slog.String("message", logMessage),
			slog.String("service", "ironic"),
		}

		if requestID != "" && requestID != "-" {
			attrs = append(attrs, slog.String("request_id", requestID))
		}
		if function != "" {
			attrs = append(attrs, slog.String("function", function))
		}
		if fileLocation != "" {
			attrs = append(attrs, slog.String("file", fileLocation))
		}

		// Map Ironic log levels to slog levels
		ctx := context.Background()
		switch level {
		case "DEBUG":
			lw.logger.LogAttrs(
				ctx,
				slog.LevelDebug,
				logMessage,
				attrs...,
			)
		case "INFO":
			lw.logger.LogAttrs(
				ctx,
				slog.LevelInfo,
				logMessage,
				attrs...,
			)
		case "WARNING", "WARN":
			lw.logger.LogAttrs(
				ctx,
				slog.LevelWarn,
				logMessage,
				attrs...,
			)
		case "ERROR":
			lw.logger.LogAttrs(
				ctx,
				slog.LevelError,
				logMessage,
				attrs...,
			)
		case "CRITICAL":
			lw.logger.LogAttrs(
				ctx,
				slog.LevelError,
				logMessage,
				attrs...,
			)
		default:
			lw.logger.LogAttrs(
				ctx,
				slog.LevelInfo,
				logMessage,
				attrs...,
			)
		}
	} else {
		// Fallback for non-matching lines
		lw.logger.Info(line)
	}
}
