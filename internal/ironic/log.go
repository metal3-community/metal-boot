package ironic

import (
	"fmt"
	"log/slog"
	"strings"
)

type logWriter struct {
	logger *slog.Logger
	prefix string
}

func (lw *logWriter) Write(p []byte) (n int, err error) {
	lines := strings.SplitSeq(strings.TrimSpace(string(p)), "\n")
	for line := range lines {
		if line != "" {
			lw.logger.Info("Log line", "line", fmt.Sprintf("%s%s", lw.prefix, line))
		}
	}
	return len(p), nil
}
