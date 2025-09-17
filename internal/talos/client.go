package talos

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/siderolabs/image-factory/pkg/client"
	"github.com/siderolabs/image-factory/pkg/schematic"
	"github.com/siderolabs/talos/pkg/machinery/platforms"
)

// Client wraps the Talos image factory client with additional functionality.
type Client struct {
	imageFactoryClient *client.Client
	logger             *slog.Logger
}

// NewClient creates a new Talos client.
func NewClient(baseURL string, logger *slog.Logger) (*Client, error) {
	imageFactoryClient, err := client.New(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create image factory client: %w", err)
	}

	return &Client{
		imageFactoryClient: imageFactoryClient,
		logger:             logger,
	}, nil
}

// GetVersions returns all available Talos versions.
func (c *Client) GetVersions(ctx context.Context) ([]string, error) {
	versions, err := c.imageFactoryClient.Versions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get talos versions: %w", err)
	}
	return versions, nil
}

// GetLatestVersion returns the latest stable Talos version.
func (c *Client) GetLatestVersion(ctx context.Context) (string, error) {
	versions, err := c.GetVersions(ctx)
	if err != nil {
		return "", err
	}

	var latest semver.Version
	var latestString string

	for _, version := range versions {
		semVer, err := semver.Parse(strings.TrimPrefix(version, "v"))
		if err != nil {
			c.logger.Warn("Failed to parse version", "version", version, "error", err)
			continue
		}

		// Skip pre-release versions for "latest"
		if len(semVer.Pre) > 0 {
			continue
		}

		if latestString == "" || semVer.GT(latest) {
			latest = semVer
			latestString = version
		}
	}

	if latestString == "" {
		return "", fmt.Errorf("no stable versions found")
	}

	return latestString, nil
}

// GetExtensionsVersions returns available extensions for a specific Talos version.
func (c *Client) GetExtensionsVersions(
	ctx context.Context,
	talosVersion string,
) ([]client.ExtensionInfo, error) {
	extensionsInfo, err := c.imageFactoryClient.ExtensionsVersions(ctx, talosVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get extensions for version %s: %w", talosVersion, err)
	}
	return extensionsInfo, nil
}

// GetFilteredExtensions returns extension details for the specified extension names.
func (c *Client) GetFilteredExtensions(
	ctx context.Context,
	talosVersion string,
	extensionNames []string,
) ([]client.ExtensionInfo, error) {
	if len(extensionNames) == 0 {
		return []client.ExtensionInfo{}, nil
	}

	extensionsInfo, err := c.imageFactoryClient.ExtensionsVersions(ctx, talosVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get extensions for version %s: %w", talosVersion, err)
	}

	// Create a map of requested extension names for fast lookup
	requestedNames := make(map[string]bool)
	for _, name := range extensionNames {
		requestedNames[name] = true
	}

	// Filter extensions to only include those requested
	var filteredExtensions []client.ExtensionInfo
	var foundNames []string

	for _, ext := range extensionsInfo {
		if requestedNames[ext.Name] {
			filteredExtensions = append(filteredExtensions, ext)
			foundNames = append(foundNames, ext.Name)
		}
	}

	// Check if all requested extensions were found
	for _, name := range extensionNames {
		found := false
		for _, foundName := range foundNames {
			if foundName == name {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf(
				"extension %q is not available for version %s",
				name,
				talosVersion,
			)
		}
	}

	return filteredExtensions, nil
}

// GetOverlaysVersions returns available overlays for a specific Talos version.
func (c *Client) GetOverlaysVersions(
	ctx context.Context,
	talosVersion string,
) ([]client.OverlayInfo, error) {
	overlaysInfo, err := c.imageFactoryClient.OverlaysVersions(ctx, talosVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get overlays for version %s: %w", talosVersion, err)
	}
	return overlaysInfo, nil
}

// CreateSchematic creates a schematic with the specified extensions and overlay.
func (c *Client) CreateSchematic(
	ctx context.Context,
	extensions []string,
	overlay string,
) (string, error) {
	schematicDef := schematic.Schematic{
		Customization: schematic.Customization{
			SystemExtensions: schematic.SystemExtensions{
				OfficialExtensions: extensions,
			},
		},
	}

	// Set overlay if specified
	if overlay != "" {
		schematicDef.Overlay = schematic.Overlay{
			Name: overlay,
		}
	}

	schematicID, err := c.imageFactoryClient.SchematicCreate(ctx, schematicDef)
	if err != nil {
		return "", fmt.Errorf("failed to create schematic: %w", err)
	}

	return schematicID, nil
}

// CreateSchematicWithDetails creates a schematic using validated extension details.
func (c *Client) CreateSchematicWithDetails(
	ctx context.Context,
	extensionDetails []client.ExtensionInfo,
	overlay string,
) (string, error) {
	// Extract extension names from the details
	extensionNames := make([]string, len(extensionDetails))
	for i, ext := range extensionDetails {
		extensionNames[i] = ext.Name
	}

	return c.CreateSchematic(ctx, extensionNames, overlay)
}

// GetImageURL returns the URL for downloading a Talos image.
func (c *Client) GetImageURL(
	ctx context.Context,
	schematicID, version, arch string,
) (string, error) {
	// Use the platforms package to get the correct path for OpenStack images
	cloudPlatforms := platforms.CloudPlatforms()
	var openstackPlatform *platforms.Platform

	for _, platform := range cloudPlatforms {
		if platform.Name == "openstack" {
			openstackPlatform = &platform
			break
		}
	}

	if openstackPlatform == nil {
		return "", fmt.Errorf("openstack platform not found")
	}

	// Generate the disk image path using the platform
	imagePath := openstackPlatform.DiskImageDefaultPath(arch)

	// Construct the full URL
	baseURL := c.imageFactoryClient.BaseURL()
	imageURL := fmt.Sprintf("%s/image/%s/%s/%s", baseURL, schematicID, version, imagePath)

	return imageURL, nil
}

// GetCompressedImageURL returns the URL for downloading a compressed Talos image.
func (c *Client) GetCompressedImageURL(
	ctx context.Context,
	schematicID, version, arch string,
) (string, error) {
	baseURL, err := c.GetImageURL(ctx, schematicID, version, arch)
	if err != nil {
		return "", err
	}

	// Replace .raw with .raw.xz for compressed version
	if strings.HasSuffix(baseURL, ".raw") {
		return strings.Replace(baseURL, ".raw", ".raw.xz", 1), nil
	}

	return baseURL + ".xz", nil
}
