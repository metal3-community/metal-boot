// Package middleware provides common TFTP middleware, ported from
// github.com/tinkerbell/tinkerbell/pkg/tftp/middleware.
package middleware

import (
	"io"
	"time"

	"github.com/go-logr/logr"
	"github.com/metal3-community/metal-boot/internal/tftp/handler"
	"github.com/pin/tftp/v3"
)

// Logging returns middleware that logs TFTP requests using the provided logger.
// It logs the filename, duration, and success/failure status.
func Logging(logger logr.Logger) func(handler.Handler) handler.Handler {
	return func(next handler.Handler) handler.Handler {
		return handler.HandlerFunc(func(filename string, rf io.ReaderFrom) error {
			start := time.Now()

			err := next.ServeTFTP(filename, rf)

			duration := time.Since(start)
			if err != nil {
				logger.Error(err, "response", "scheme", "tftp", "method", "GET", "uri", filename, "duration", duration, "status", "failure")
			} else {
				logger.Info("response", "scheme", "tftp", "method", "GET", "uri", filename, "duration", duration, "status", "success")
			}
			return err
		})
	}
}

// transferLogging implements tftp.Hook for logging TFTP transfer statistics.
type transferLogging struct {
	log logr.Logger
}

// NewTransferHook returns a tftp.Hook that logs transfer statistics.
func NewTransferHook(log logr.Logger) tftp.Hook {
	return &transferLogging{log: log}
}

// statsFields flattens TransferStats into logr key-value pairs.
func statsFields(stats *tftp.TransferStats) []any {
	return []any{
		"filename", stats.Filename,
		"remoteAddr", stats.RemoteAddr.String(),
		"duration", stats.Duration.String(),
		"datagramsSent", stats.DatagramsSent,
		"datagramsAcked", stats.DatagramsAcked,
		"mode", stats.Mode,
		"tid", stats.Tid,
	}
}

// OnSuccess logs successful TFTP transfers.
func (h *transferLogging) OnSuccess(stats tftp.TransferStats) {
	h.log.Info("tftp transfer successful", statsFields(&stats)...)
}

// OnFailure logs failed TFTP transfers.
func (h *transferLogging) OnFailure(stats tftp.TransferStats, err error) {
	h.log.Error(err, "tftp transfer failed", statsFields(&stats)...)
}
