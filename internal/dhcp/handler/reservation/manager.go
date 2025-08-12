package reservation

import (
	"fmt"

	"github.com/bmcpi/pibmc/internal/dhcp/lease"
)

// NewHandlerWithLeaseManagement creates a new Handler with lease and config management.
func NewHandlerWithLeaseManagement(
	handler *Handler,
	leaseFile, configFile string,
) (*Handler, error) {
	// Initialize lease manager
	if leaseFile != "" {
		handler.LeaseManager = lease.NewManager(leaseFile)
		if err := handler.LeaseManager.LoadLeases(); err != nil {
			return nil, fmt.Errorf("failed to load DHCP leases: %w", err)
		}
	}

	// Initialize config manager
	if configFile != "" {
		handler.ConfigManager = lease.NewConfigManager(configFile)
		if err := handler.ConfigManager.LoadConfig(); err != nil {
			return nil, fmt.Errorf("failed to load DHCP configuration: %w", err)
		}
	}

	return handler, nil
}

// CleanupExpiredLeases removes expired leases from the lease manager.
// This should be called periodically to clean up old leases.
func (h *Handler) CleanupExpiredLeases() {
	if h.LeaseManager != nil {
		h.LeaseManager.CleanExpiredLeases()

		// Save cleaned leases to file
		if err := h.LeaseManager.SaveLeases(); err != nil {
			h.Log.Error(err, "failed to save cleaned DHCP leases")
		}
	}
}
