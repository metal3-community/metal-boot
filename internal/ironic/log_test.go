package ironic

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

// testLogHandler captures slog output for testing.
type testLogHandler struct {
	logs []testLogEntry
}

type testLogEntry struct {
	Level   slog.Level
	Message string
	Attrs   map[string]any
}

func (h *testLogHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *testLogHandler) Handle(_ context.Context, record slog.Record) error {
	attrs := make(map[string]any)
	record.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	h.logs = append(h.logs, testLogEntry{
		Level:   record.Level,
		Message: record.Message,
		Attrs:   attrs,
	})
	return nil
}

func (h *testLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *testLogHandler) WithGroup(name string) slog.Handler {
	return h
}

func TestLogWriter_ParseIronicLogs(t *testing.T) {
	tests := []struct {
		name           string
		logLine        string
		expectedLevel  slog.Level
		expectedAttrs  map[string]string
		expectedPrefix string
	}{
		{
			name:          "debug log with function and file",
			logLine:       "2025-08-01 19:46:07.068 1 DEBUG sushy.resources.base [None req-6a31fd4b-8574-4c48-859f-9e00972eee1b - - - - - -] Received representation of System /redfish/v1/Systems/System.Embedded.1 parse_attributes /usr/lib/python3.13/site-packages/sushy/resources/base.py:112",
			expectedLevel: slog.LevelDebug,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-01 19:46:07.068",
				"process_id": "1",
				"module":     "sushy.resources.base",
				"request_id": "None req-6a31fd4b-8574-4c48-859f-9e00972eee1b - - - - - -",
				"message":    "Received representation of System /redfish/v1/Systems/System.Embedded.1",
				"function":   "parse_attributes",
				"file":       "/usr/lib/python3.13/site-packages/sushy/resources/base.py:112",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "debug log with cleanwait timeouts",
			logLine:       "2025-08-01 19:46:08.125 1 DEBUG futurist.periodics [-] Submitting periodic callback 'ironic.conductor.manager.ConductorManager._check_cleanwait_timeouts' _process_scheduled /usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			expectedLevel: slog.LevelDebug,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-01 19:46:08.125",
				"process_id": "1",
				"module":     "futurist.periodics",
				"message":    "Submitting periodic callback 'ironic.conductor.manager.ConductorManager._check_cleanwait_timeouts'",
				"function":   "_process_scheduled",
				"file":       "/usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "info level log",
			logLine:       "2025-08-01 19:46:09.211 1 INFO ironic.conductor.manager [-] Starting conductor service",
			expectedLevel: slog.LevelInfo,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-01 19:46:09.211",
				"process_id": "1",
				"module":     "ironic.conductor.manager",
				"message":    "Starting conductor service",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "warning level log",
			logLine:       "2025-08-01 19:46:09.311 1 WARNING ironic.api.wsgi [req-123] Deprecated API endpoint used",
			expectedLevel: slog.LevelWarn,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-01 19:46:09.311",
				"process_id": "1",
				"module":     "ironic.api.wsgi",
				"request_id": "req-123",
				"message":    "Deprecated API endpoint used",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "error level log",
			logLine:       "2025-08-01 19:46:09.411 1 ERROR ironic.conductor.task [req-456] Failed to deploy node",
			expectedLevel: slog.LevelError,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-01 19:46:09.411",
				"process_id": "1",
				"module":     "ironic.conductor.task",
				"request_id": "req-456",
				"message":    "Failed to deploy node",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "critical level log",
			logLine:       "2025-08-01 19:46:09.511 1 CRITICAL ironic.conductor.manager [-] Critical system failure",
			expectedLevel: slog.LevelError,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-01 19:46:09.511",
				"process_id": "1",
				"module":     "ironic.conductor.manager",
				"message":    "Critical system failure",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "log with complex message and no file info",
			logLine:       "2025-08-01 19:46:09.611 1 INFO ironic.drivers.modules.agent [req-789] Agent heartbeat received for node 12345-abcd with status 'clean wait'",
			expectedLevel: slog.LevelInfo,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-01 19:46:09.611",
				"process_id": "1",
				"module":     "ironic.drivers.modules.agent",
				"request_id": "req-789",
				"message":    "Agent heartbeat received for node 12345-abcd with status 'clean wait'",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "complex request id with power state sync",
			logLine:       "2025-08-01 19:46:09.711 1 DEBUG ironic.conductor.manager [None req-e92ad424-d007-41b1-942c-3df0c8583a9b - - - - - -] Completed power state sync operation, evaluated 1 nodes. _sync_power_states /usr/lib/python3.13/site-packages/ironic/conductor/manager.py:1581",
			expectedLevel: slog.LevelDebug,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-01 19:46:09.711",
				"process_id": "1",
				"module":     "ironic.conductor.manager",
				"request_id": "None req-e92ad424-d007-41b1-942c-3df0c8583a9b - - - - - -",
				"message":    "Completed power state sync operation, evaluated 1 nodes.",
				"function":   "_sync_power_states",
				"file":       "/usr/lib/python3.13/site-packages/ironic/conductor/manager.py:1581",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "redfish firmware query update failed",
			logLine:       "2025-08-01 19:46:09.811 1 DEBUG futurist.periodics [-] Submitting periodic callback 'ironic.drivers.modules.redfish.firmware.RedfishFirmware._query_update_failed' _process_scheduled /usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			expectedLevel: slog.LevelDebug,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-01 19:46:09.811",
				"process_id": "1",
				"module":     "futurist.periodics",
				"message":    "Submitting periodic callback 'ironic.drivers.modules.redfish.firmware.RedfishFirmware._query_update_failed'",
				"function":   "_process_scheduled",
				"file":       "/usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "redfish firmware query update status",
			logLine:       "2025-08-01 19:46:09.912 1 DEBUG futurist.periodics [-] Submitting periodic callback 'ironic.drivers.modules.redfish.firmware.RedfishFirmware._query_update_status' _process_scheduled /usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			expectedLevel: slog.LevelDebug,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-01 19:46:09.912",
				"process_id": "1",
				"module":     "futurist.periodics",
				"message":    "Submitting periodic callback 'ironic.drivers.modules.redfish.firmware.RedfishFirmware._query_update_status'",
				"function":   "_process_scheduled",
				"file":       "/usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "conductor periodics async firmware update failed",
			logLine:       "2025-08-01 19:46:10.015 1 DEBUG ironic.conductor.periodics [-] Completed periodic task for purpose checking if async update of firmware component failed. wrapper /usr/lib/python3.13/site-packages/ironic/conductor/periodics.py:174",
			expectedLevel: slog.LevelDebug,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-01 19:46:10.015",
				"process_id": "1",
				"module":     "ironic.conductor.periodics",
				"message":    "Completed periodic task for purpose checking if async update of firmware component failed.",
				"function":   "wrapper",
				"file":       "/usr/lib/python3.13/site-packages/ironic/conductor/periodics.py:174",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "conductor periodics async firmware update",
			logLine:       "2025-08-27 05:03:58.006 16 DEBUG ironic.conductor.periodics [-] Completed periodic task for purpose checking async update of firmware component. wrapper /usr/lib/python3.13/site-packages/ironic/conductor/periodics.py:174",
			expectedLevel: slog.LevelDebug,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-27 05:03:58.006",
				"process_id": "16",
				"module":     "ironic.conductor.periodics",
				"message":    "Completed periodic task for purpose checking async update of firmware component.",
				"function":   "wrapper",
				"file":       "/usr/lib/python3.13/site-packages/ironic/conductor/periodics.py:174",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "redfish management query firmware update failed",
			logLine:       "2025-08-27 05:03:58.007 16 DEBUG futurist.periodics [-] Submitting periodic callback 'ironic.drivers.modules.redfish.management.RedfishManagement._query_firmware_update_failed' _process_scheduled /usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			expectedLevel: slog.LevelDebug,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-27 05:03:58.007",
				"process_id": "16",
				"module":     "futurist.periodics",
				"message":    "Submitting periodic callback 'ironic.drivers.modules.redfish.management.RedfishManagement._query_firmware_update_failed'",
				"function":   "_process_scheduled",
				"file":       "/usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "drac management import configuration status",
			logLine:       "2025-08-27 05:03:58.014 16 DEBUG futurist.periodics [-] Submitting periodic callback 'ironic.drivers.modules.drac.management.DracRedfishManagement._query_import_configuration_status' _process_scheduled /usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			expectedLevel: slog.LevelDebug,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-27 05:03:58.014",
				"process_id": "16",
				"module":     "futurist.periodics",
				"message":    "Submitting periodic callback 'ironic.drivers.modules.drac.management.DracRedfishManagement._query_import_configuration_status'",
				"function":   "_process_scheduled",
				"file":       "/usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "redfish raid query config failed",
			logLine:       "2025-08-27 05:03:58.030 16 DEBUG futurist.periodics [-] Submitting periodic callback 'ironic.drivers.modules.redfish.raid.RedfishRAID._query_raid_config_failed' _process_scheduled /usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			expectedLevel: slog.LevelDebug,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-27 05:03:58.030",
				"process_id": "16",
				"module":     "futurist.periodics",
				"message":    "Submitting periodic callback 'ironic.drivers.modules.redfish.raid.RedfishRAID._query_raid_config_failed'",
				"function":   "_process_scheduled",
				"file":       "/usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "redfish raid query config status",
			logLine:       "2025-08-27 05:03:58.033 16 DEBUG futurist.periodics [-] Submitting periodic callback 'ironic.drivers.modules.redfish.raid.RedfishRAID._query_raid_config_status' _process_scheduled /usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			expectedLevel: slog.LevelDebug,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-27 05:03:58.033",
				"process_id": "16",
				"module":     "futurist.periodics",
				"message":    "Submitting periodic callback 'ironic.drivers.modules.redfish.raid.RedfishRAID._query_raid_config_status'",
				"function":   "_process_scheduled",
				"file":       "/usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "conductor periodics async raid config failed",
			logLine:       "2025-08-27 05:03:58.036 16 DEBUG ironic.conductor.periodics [-] Completed periodic task for purpose checking async RAID config failed. wrapper /usr/lib/python3.13/site-packages/ironic/conductor/periodics.py:174",
			expectedLevel: slog.LevelDebug,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-27 05:03:58.036",
				"process_id": "16",
				"module":     "ironic.conductor.periodics",
				"message":    "Completed periodic task for purpose checking async RAID config failed.",
				"function":   "wrapper",
				"file":       "/usr/lib/python3.13/site-packages/ironic/conductor/periodics.py:174",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "pxe base mixin check boot timeouts",
			logLine:       "2025-08-27 05:04:27.899 16 DEBUG futurist.periodics [-] Submitting periodic callback 'ironic.drivers.modules.pxe_base.PXEBaseMixin._check_boot_timeouts' _process_scheduled /usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			expectedLevel: slog.LevelDebug,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-27 05:04:27.899",
				"process_id": "16",
				"module":     "futurist.periodics",
				"message":    "Submitting periodic callback 'ironic.drivers.modules.pxe_base.PXEBaseMixin._check_boot_timeouts'",
				"function":   "_process_scheduled",
				"file":       "/usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "conductor periodics pxe boot status",
			logLine:       "2025-08-27 05:04:27.905 16 DEBUG ironic.conductor.periodics [-] Completed periodic task for purpose checking PXE boot status. wrapper /usr/lib/python3.13/site-packages/ironic/conductor/periodics.py:174",
			expectedLevel: slog.LevelDebug,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-27 05:04:27.905",
				"process_id": "16",
				"module":     "ironic.conductor.periodics",
				"message":    "Completed periodic task for purpose checking PXE boot status.",
				"function":   "wrapper",
				"file":       "/usr/lib/python3.13/site-packages/ironic/conductor/periodics.py:174",
			},
			expectedPrefix: "[test] ",
		},
		{
			name:          "actual failing log line",
			logLine:       "2025-08-27 05:11:34.670 15 DEBUG futurist.periodics [-] Submitting periodic callback 'ironic.drivers.modules.pxe_base.PXEBaseMixin._check_boot_timeouts' _process_scheduled /usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			expectedLevel: slog.LevelDebug,
			expectedAttrs: map[string]string{
				"timestamp":  "2025-08-27 05:11:34.670",
				"process_id": "15",
				"module":     "futurist.periodics",
				"message":    "Submitting periodic callback 'ironic.drivers.modules.pxe_base.PXEBaseMixin._check_boot_timeouts'",
				"function":   "_process_scheduled",
				"file":       "/usr/lib/python3.13/site-packages/futurist/periodics.py:638",
			},
			expectedPrefix: "[test] ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			handler := &testLogHandler{}
			logger := slog.New(handler)

			// Create logWriter with test prefix
			lw := &logWriter{
				logger: logger,
				prefix: tt.expectedPrefix,
			}

			// Write the log line
			_, err := lw.Write([]byte(tt.logLine))
			if err != nil {
				t.Fatalf("Write() error = %v", err)
			}

			// Verify we got exactly one log entry
			if len(handler.logs) != 1 {
				t.Fatalf("Expected 1 log entry, got %d", len(handler.logs))
			}

			log := handler.logs[0]

			// Check log level
			if log.Level != tt.expectedLevel {
				t.Errorf("Expected level %v, got %v", tt.expectedLevel, log.Level)
			}

			// Check message (should include prefix)
			expectedMessage := tt.expectedPrefix + tt.expectedAttrs["message"]
			if log.Message != expectedMessage {
				t.Errorf("Expected message %q, got %q", expectedMessage, log.Message)
			}

			// Check attributes
			for key, expectedValue := range tt.expectedAttrs {
				if key == "message" {
					continue // Already checked in message
				}

				actualValue, exists := log.Attrs[key]
				if !exists {
					t.Errorf("Missing attribute %q", key)
					continue
				}

				if actualValue != expectedValue {
					t.Errorf("Attribute %q: expected %q, got %q", key, expectedValue, actualValue)
				}
			}

			// Verify no extra attributes (except expected ones)
			// If no request_id in expected, it shouldn't be in actual either
			if _, hasRequestID := tt.expectedAttrs["request_id"]; !hasRequestID {
				if _, exists := log.Attrs["request_id"]; exists {
					t.Errorf("Unexpected request_id attribute found")
				}
			}

			// Check for function and file attributes if expected
			if _, hasFunction := tt.expectedAttrs["function"]; !hasFunction {
				if _, exists := log.Attrs["function"]; exists {
					t.Errorf("Unexpected function attribute found")
				}
			}

			if _, hasFile := tt.expectedAttrs["file"]; !hasFile {
				if _, exists := log.Attrs["file"]; exists {
					t.Errorf("Unexpected file attribute found")
				}
			}
		})
	}
}

func TestLogWriter_NonMatchingLines(t *testing.T) {
	handler := &testLogHandler{}
	logger := slog.New(handler)

	lw := &logWriter{
		logger: logger,
		prefix: "[test] ",
	}

	// Test non-matching log line
	nonMatchingLine := "This is not an Ironic log line"
	_, err := lw.Write([]byte(nonMatchingLine))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Should have one log entry with raw line
	if len(handler.logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(handler.logs))
	}

	log := handler.logs[0]

	// Should be logged as info level
	if log.Level != slog.LevelInfo {
		t.Errorf("Expected level %v, got %v", slog.LevelInfo, log.Level)
	}

	// Should have "Raw log line" message
	if log.Message != "Raw log line" {
		t.Errorf("Expected message 'Raw log line', got %q", log.Message)
	}

	// Should have line attribute with prefixed content
	expectedLine := "[test] " + nonMatchingLine
	if log.Attrs["line"] != expectedLine {
		t.Errorf("Expected line attribute %q, got %q", expectedLine, log.Attrs["line"])
	}
}

func TestLogWriter_MultipleLines(t *testing.T) {
	handler := &testLogHandler{}
	logger := slog.New(handler)

	lw := &logWriter{
		logger: logger,
		prefix: "[test] ",
	}

	// Test multiple lines in one write
	multiLineInput := strings.Join([]string{
		"2025-08-27 04:53:57.911 16 DEBUG futurist.periodics [-] First message",
		"2025-08-27 04:53:57.912 16 INFO ironic.conductor.manager [-] Second message",
		"", // Empty line should be ignored
		"2025-08-27 04:53:57.913 16 ERROR ironic.api.wsgi [-] Third message",
	}, "\n")

	_, err := lw.Write([]byte(multiLineInput))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Should have three log entries (empty line ignored)
	if len(handler.logs) != 3 {
		t.Fatalf("Expected 3 log entries, got %d", len(handler.logs))
	}

	// Check first log (DEBUG)
	if handler.logs[0].Level != slog.LevelDebug {
		t.Errorf("First log: expected level %v, got %v", slog.LevelDebug, handler.logs[0].Level)
	}

	// Check second log (INFO)
	if handler.logs[1].Level != slog.LevelInfo {
		t.Errorf("Second log: expected level %v, got %v", slog.LevelInfo, handler.logs[1].Level)
	}

	// Check third log (ERROR)
	if handler.logs[2].Level != slog.LevelError {
		t.Errorf("Third log: expected level %v, got %v", slog.LevelError, handler.logs[2].Level)
	}
}

func TestLogWriter_RegexPattern(t *testing.T) {
	// Test the regex pattern directly
	testCases := []struct {
		input    string
		expected bool
	}{
		{
			input:    "2025-08-27 04:53:57.911 16 DEBUG futurist.periodics [-] Test message",
			expected: true,
		},
		{
			input:    "2025-08-27 04:53:57.911 16 INFO ironic.conductor [-] Test message",
			expected: true,
		},
		{
			input:    "2025-08-27 04:53:57.911 16 ERROR ironic.api [req-123] Test message",
			expected: true,
		},
		{
			input:    "Not an Ironic log line",
			expected: false,
		},
		{
			input:    "Missing timestamp",
			expected: false,
		},
		{
			input:    "2025-08-27 04:53:57.911 DEBUG missing_pid",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			matches := ironicLogPattern.FindStringSubmatch(tc.input)
			matched := len(matches) == 7

			if matched != tc.expected {
				t.Errorf(
					"Pattern match for %q: expected %v, got %v",
					tc.input,
					tc.expected,
					matched,
				)
			}

			if matched && tc.expected {
				// Additional validation for successful matches
				timestamp := matches[1]
				processID := matches[2]
				level := matches[3]
				module := matches[4]
				requestID := matches[5]
				message := matches[6]

				if timestamp == "" || processID == "" || level == "" || module == "" ||
					message == "" {
					t.Errorf(
						"Empty field in match: timestamp=%q, processID=%q, level=%q, module=%q, requestID=%q, message=%q",
						timestamp,
						processID,
						level,
						module,
						requestID,
						message,
					)
				}
			}
		})
	}
}
