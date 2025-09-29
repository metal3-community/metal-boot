package ironic

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/metal3-community/metal-boot/internal/util"
)

func TestProcessManager_SharedPath(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx := context.Background()

	tests := []struct {
		name       string
		sharedRoot string
		path       string
		want       string
	}{
		{
			name:       "default shared root",
			sharedRoot: "",
			path:       "/log/ironic/ironic.log",
			want:       "/shared/log/ironic/ironic.log",
		},
		{
			name:       "custom shared root",
			sharedRoot: "/custom",
			path:       "/log/ironic/ironic.log",
			want:       "/custom/log/ironic/ironic.log",
		},
		{
			name:       "custom shared root with trailing slash",
			sharedRoot: "/custom/",
			path:       "/log/ironic/ironic.log",
			want:       "/custom//log/ironic/ironic.log",
		},
		{
			name:       "relative path",
			sharedRoot: "/opt/shared",
			path:       "/data/images",
			want:       "/opt/shared/data/images",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				SharedRoot: tt.sharedRoot,
			}
			pm := NewProcessManager(ctx, logger, config)

			got := pm.sharedPath(tt.path)
			if got != tt.want {
				t.Errorf("ProcessManager.sharedPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessManager_DefaultSharedRoot(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx := context.Background()

	t.Run("default shared root is set when empty", func(t *testing.T) {
		config := &Config{}
		pm := NewProcessManager(ctx, logger, config)

		if config.SharedRoot != "/shared" {
			t.Errorf("Expected SharedRoot to be '/shared', got '%s'", config.SharedRoot)
		}

		// Test that sharedPath works with the default
		got := pm.sharedPath("/log/ironic/ironic.log")
		want := "/shared/log/ironic/ironic.log"
		if got != want {
			t.Errorf("ProcessManager.sharedPath() = %v, want %v", got, want)
		}
	})

	t.Run("custom shared root is preserved", func(t *testing.T) {
		config := &Config{
			SharedRoot: "/custom/shared",
		}
		pm := NewProcessManager(ctx, logger, config)

		if config.SharedRoot != "/custom/shared" {
			t.Errorf("Expected SharedRoot to be '/custom/shared', got '%s'", config.SharedRoot)
		}

		// Test that sharedPath works with the custom root
		got := pm.sharedPath("/log/ironic/ironic.log")
		want := "/custom/shared/log/ironic/ironic.log"
		if got != want {
			t.Errorf("ProcessManager.sharedPath() = %v, want %v", got, want)
		}
	})
}

func TestProcessManager_GenerateIronicConfigWithSharedRoot(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx := context.Background()

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "ironic.conf")

	tests := []struct {
		name        string
		sharedRoot  string
		wantLogFile string
	}{
		{
			name:        "default shared root",
			sharedRoot:  "",
			wantLogFile: "/shared/log/ironic/ironic.log",
		},
		{
			name:        "custom shared root",
			sharedRoot:  "/opt/shared",
			wantLogFile: "/opt/shared/log/ironic/ironic.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				SharedRoot: tt.sharedRoot,
				ConfigPath: configPath,
				SocketPath: filepath.Join(tempDir, "ironic.sock"),
			}

			pm := NewProcessManager(ctx, logger, config)

			// Generate the config
			err := pm.generateIronicConfig()
			if err != nil {
				t.Fatalf("generateIronicConfig() failed: %v", err)
			}

			// Verify the log file path is correct
			if pm.config.Default.LogFile != tt.wantLogFile {
				t.Errorf(
					"Expected LogFile to be '%s', got '%s'",
					tt.wantLogFile,
					pm.config.Default.LogFile,
				)
			}

			// Verify the config file was created
			if _, err := os.Stat(configPath); err != nil {
				t.Errorf("Config file was not created: %v", err)
			}
		})
	}
}

func TestConfig_SetDefaults(t *testing.T) {
	t.Run("sets all default values correctly", func(t *testing.T) {
		config := &Config{}
		config.SetDefaults()

		// Verify some key defaults are set
		if config.Default.AuthStrategy != "noauth" {
			t.Errorf("Expected AuthStrategy to be 'noauth', got '%s'", config.Default.AuthStrategy)
		}
		if config.Default.Debug == nil || !*config.Default.Debug {
			t.Errorf("Expected Debug to be true, got %v", config.Default.Debug)
		}
		if config.Default.RPCTransport != "none" {
			t.Errorf(
				"Expected RPCTransport to be 'none', got '%s'",
				config.Default.RPCTransport,
			)
		}
		// These runtime-dependent paths should NOT be set by SetDefaults
		if config.Agent.DeployLogsLocalPath != "" {
			t.Errorf(
				"Expected DeployLogsLocalPath to be empty (set by SetRuntimePaths), got '%s'",
				config.Agent.DeployLogsLocalPath,
			)
		}
		if config.Database.Connection != "sqlite:////var/lib/ironic/ironic.db" {
			t.Errorf(
				"Expected Database Connection to be 'sqlite:////var/lib/ironic/ironic.db', got '%s'",
				config.Database.Connection,
			)
		}
		if config.DHCP.DHCPProvider != "none" {
			t.Errorf("Expected DHCPProvider to be 'none', got '%s'", config.DHCP.DHCPProvider)
		}
	})

	t.Run("does not override existing values", func(t *testing.T) {
		config := &Config{
			Default: DefaultConfig{
				AuthStrategy: "custom",
				Debug:        util.Ptr(false),
			},
			Database: DatabaseConfig{
				Connection: "custom://connection",
			},
		}
		config.SetDefaults()

		// Verify existing values are preserved
		if config.Default.AuthStrategy != "custom" {
			t.Errorf(
				"Expected AuthStrategy to remain 'custom', got '%s'",
				config.Default.AuthStrategy,
			)
		}
		if config.Default.Debug == nil || *config.Default.Debug {
			t.Errorf("Expected Debug to remain false, got %v", config.Default.Debug)
		}
		if config.Database.Connection != "custom://connection" {
			t.Errorf(
				"Expected Database Connection to remain 'custom://connection', got '%s'",
				config.Database.Connection,
			)
		}

		// Verify other defaults are still set
		if config.Default.RPCTransport != "none" {
			t.Errorf(
				"Expected RPCTransport to be 'none', got '%s'",
				config.Default.RPCTransport,
			)
		}
	})
}

func TestConfig_SetRuntimePaths(t *testing.T) {
	t.Run("sets runtime paths correctly", func(t *testing.T) {
		config := &Config{}
		config.SetRuntimePaths("/tmp/test.sock", "/custom/shared")

		if config.Default.LogFile != "/custom/shared/log/ironic/ironic.log" {
			t.Errorf(
				"Expected LogFile to be '/custom/shared/log/ironic/ironic.log', got '%s'",
				config.Default.LogFile,
			)
		}
		if config.API.UnixSocket != "/tmp/test.sock" {
			t.Errorf("Expected UnixSocket to be '/tmp/test.sock', got '%s'", config.API.UnixSocket)
		}
		if config.Conductor.APIURL != "http+unix:///tmp/test.sock" {
			t.Errorf(
				"Expected APIURL to be 'http+unix:///tmp/test.sock', got '%s'",
				config.Conductor.APIURL,
			)
		}

		// Test the new shared root dependent paths
		if config.Agent.DeployLogsLocalPath != "/custom/shared/log/ironic/deploy" {
			t.Errorf(
				"Expected DeployLogsLocalPath to be '/custom/shared/log/ironic/deploy', got '%s'",
				config.Agent.DeployLogsLocalPath,
			)
		}
		if config.PXE.ImagesPath != "/custom/shared/html/tmp" {
			t.Errorf(
				"Expected ImagesPath to be '/custom/shared/html/tmp', got '%s'",
				config.PXE.ImagesPath,
			)
		}
		if config.PXE.TFTPRoot != "/custom/shared/tftpboot" {
			t.Errorf(
				"Expected TFTPRoot to be '/custom/shared/tftpboot', got '%s'",
				config.PXE.TFTPRoot,
			)
		}
		if config.Deploy.HTTPRoot != "/custom/shared/html/" {
			t.Errorf(
				"Expected HTTPRoot to be '/custom/shared/html/', got '%s'",
				config.Deploy.HTTPRoot,
			)
		}
	})

	t.Run("does not override existing values", func(t *testing.T) {
		config := &Config{
			Default: DefaultConfig{
				LogFile: "/existing/log.log",
			},
			API: APIConfig{
				UnixSocket: "/existing/socket",
			},
			PXE: PXEConfig{
				ImagesPath: "/existing/images",
				TFTPRoot:   "/existing/tftp",
			},
		}
		config.SetRuntimePaths("/tmp/test.sock", "/custom/shared")

		// Verify existing values are preserved
		if config.Default.LogFile != "/existing/log.log" {
			t.Errorf(
				"Expected LogFile to remain '/existing/log.log', got '%s'",
				config.Default.LogFile,
			)
		}
		if config.API.UnixSocket != "/existing/socket" {
			t.Errorf(
				"Expected UnixSocket to remain '/existing/socket', got '%s'",
				config.API.UnixSocket,
			)
		}
		if config.PXE.ImagesPath != "/existing/images" {
			t.Errorf(
				"Expected ImagesPath to remain '/existing/images', got '%s'",
				config.PXE.ImagesPath,
			)
		}
		if config.PXE.TFTPRoot != "/existing/tftp" {
			t.Errorf("Expected TFTPRoot to remain '/existing/tftp', got '%s'", config.PXE.TFTPRoot)
		}

		// Verify other runtime paths are still set
		if config.Conductor.APIURL != "http+unix:///tmp/test.sock" {
			t.Errorf(
				"Expected APIURL to be 'http+unix:///tmp/test.sock', got '%s'",
				config.Conductor.APIURL,
			)
		}
	})
}

func TestProcessManager_IntegrationWithDefaults(t *testing.T) {
	t.Run("NewProcessManager applies defaults correctly", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		ctx := context.Background()

		// Start with minimal config (like main.go would provide)
		config := &Config{
			// Only the fields that would come from external configuration
			API: APIConfig{
				Port:           6385,
				PublicEndpoint: "http://localhost:6385",
			},
			Database: DatabaseConfig{
				Connection: "sqlite:////custom/ironic.db",
			},
		}

		pm := NewProcessManager(ctx, logger, config)
		_ = pm // We don't need to use pm, just need it to call NewProcessManager

		// Verify that defaults were applied
		if config.Default.AuthStrategy != "noauth" {
			t.Errorf(
				"Expected AuthStrategy default 'noauth', got '%s'",
				config.Default.AuthStrategy,
			)
		}
		if config.Agent.DeployLogsLocalPath != "/shared/log/ironic/deploy" {
			t.Errorf(
				"Expected DeployLogsLocalPath default, got '%s'",
				config.Agent.DeployLogsLocalPath,
			)
		}

		// Verify that runtime paths were set
		if config.Default.LogFile != "/shared/log/ironic/ironic.log" {
			t.Errorf("Expected LogFile to be set to shared path, got '%s'", config.Default.LogFile)
		}
		if config.API.UnixSocket != "/tmp/ironic.sock" {
			t.Errorf("Expected UnixSocket to be set to default, got '%s'", config.API.UnixSocket)
		}

		// Verify that external config values were preserved
		if config.API.Port != 6385 {
			t.Errorf("Expected Port to be preserved as 6385, got %d", config.API.Port)
		}
		if config.Database.Connection != "sqlite:////custom/ironic.db" {
			t.Errorf(
				"Expected custom Database Connection to be preserved, got '%s'",
				config.Database.Connection,
			)
		}
	})
}
