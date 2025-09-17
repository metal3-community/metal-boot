# Talos Image Handler Implementation Instructions

---
applyTo: 'api/images/talos/**'
---

## Overview

This document provides detailed instructions for implementing an optionally enabled Talos image handler that provides raw Talos OpenStack images varying by architecture, with parameters for Talos image factory extensions and overlays.

## API Specification

### Endpoint Pattern
```
GET /images/talos/{talos_version}/{arch}/openstack.raw?extensions={extensions}&overlay={overlay}
```

### Parameters
- `talos_version`: Specific version (e.g., "v1.11.1") or "latest"
- `arch`: Architecture ("amd64" or "arm64")
- `extensions`: Comma-separated list of extension names (optional)
- `overlay`: Single overlay name (optional)

### Example URLs
```
### Example URLs
```bash
/images/talos/latest/amd64/openstack.raw
/images/talos/v1.11.1/amd64/openstack.raw?extensions=qemu-guest-agent,nvidia-open-gpu-kernel-modules
/images/talos/v1.11.1/arm64/openstack.raw?overlay=rpi-generic
```
```

## Implementation Checklist

### Phase 1: Configuration and Structure
- [ ] Add `TalosConfig` struct to `internal/config/config.go`
  - [ ] `Enabled bool` - Enable/disable Talos handler
  - [ ] `BaseURL string` - Talos image factory base URL (default: "https://factory.talos.dev")
  - [ ] `CacheDirectory string` - Local cache directory for images
  - [ ] `MaxCacheSize int64` - Maximum cache size in bytes (optional)
  - [ ] `DefaultExtensions []string` - Default extensions to include
- [ ] Update main `Config` struct to include `Talos TalosConfig`
- [ ] Add configuration example to `configs/config.example.yaml`
- [ ] Add environment variable mapping (TALOS_ENABLED, TALOS_BASE_URL, etc.)

### Phase 2: Core Client Implementation
- [ ] Add required dependencies to `go.mod`:
  - [ ] `github.com/blang/semver/v4` - Semantic version parsing
  - [ ] `github.com/siderolabs/image-factory/pkg/client` - Talos image factory client
  - [ ] `github.com/siderolabs/image-factory/pkg/schematic` - Schematic definitions
  - [ ] `github.com/siderolabs/talos/pkg/machinery/platforms` - Platform definitions
  - [ ] Standard library: `compress/gzip`, `crypto/sha256`, `encoding/json`
- [ ] Create `internal/talos/` package with wrapper functionality
- [ ] Implement `Client` struct wrapping the image factory client:
  ```go
  import (
      "github.com/siderolabs/image-factory/pkg/client"
      "github.com/siderolabs/image-factory/pkg/schematic"
      "github.com/siderolabs/talos/pkg/machinery/platforms"
  )

  type Client struct {
      imageFactoryClient *client.Client
      logger             *slog.Logger
  }

  func NewClient(baseURL string, logger *slog.Logger) (*Client, error) {
      imageFactoryClient, err := client.NewClient(baseURL)
      if err != nil {
          return nil, fmt.Errorf("failed to create image factory client: %w", err)
      }
      
      return &Client{
          imageFactoryClient: imageFactoryClient,
          logger:             logger,
      }, nil
  }
  ```

### Phase 3: Image Factory API Integration
Based on analysis of siderolabs/terraform-provider-talos implementation:

#### Version Management
- [ ] Implement `GetVersions(ctx context.Context) ([]string, error)` using the image factory client:
  ```go
  func (c *Client) GetVersions(ctx context.Context) ([]string, error) {
      versions, err := c.imageFactoryClient.Versions(ctx)
      if err != nil {
          return nil, fmt.Errorf("failed to get talos versions: %w", err)
      }
      return versions, nil
  }
  ```
- [ ] Implement `GetLatestVersion(ctx context.Context) (string, error)` with semantic version parsing:
  ```go
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
  ```

#### Extension/Overlay Management  
- [ ] Use `client.ExtensionInfo` struct from the image factory package (pre-defined):
  ```go
  // client.ExtensionInfo is already defined in the package with:
  // type ExtensionInfo struct {
  //     Name        string
  //     Ref         string  
  //     Digest      string
  //     Author      string
  //     Description string
  // }
  ```
- [ ] Implement `GetExtensionsVersions(ctx context.Context, talosVersion string) ([]client.ExtensionInfo, error)`:
  ```go
  func (c *Client) GetExtensionsVersions(ctx context.Context, talosVersion string) ([]client.ExtensionInfo, error) {
      extensionsInfo, err := c.imageFactoryClient.ExtensionsVersions(ctx, talosVersion)
      if err != nil {
          return nil, fmt.Errorf("failed to get extensions for version %s: %w", talosVersion, err)
      }
      return extensionsInfo, nil
  }
  ```
- [ ] Implement `GetOverlaysVersions(ctx context.Context, talosVersion string) ([]client.OverlayInfo, error)`:
  ```go
  func (c *Client) GetOverlaysVersions(ctx context.Context, talosVersion string) ([]client.OverlayInfo, error) {
      overlaysInfo, err := c.imageFactoryClient.OverlaysVersions(ctx, talosVersion)
      if err != nil {
          return nil, fmt.Errorf("failed to get overlays for version %s: %w", talosVersion, err)
      }
      return overlaysInfo, nil
  }
  ```

#### Schematic Creation
- [ ] Use the `schematic.Schematic` struct from the image factory package:
  ```go
  func (c *Client) CreateSchematic(ctx context.Context, extensions, overlays []string) (string, error) {
      schematicDef := schematic.Schematic{
          Customization: schematic.CustomizationSpec{
              SystemExtensions: make([]schematic.SystemExtension, len(extensions)),
          },
      }
      
      // Add extensions
      for i, extension := range extensions {
          schematicDef.Customization.SystemExtensions[i] = schematic.SystemExtension{
              Name: extension,
          }
      }
      
      // Add overlays to SystemExtensions (they're treated similarly)
      for _, overlay := range overlays {
          schematicDef.Customization.SystemExtensions = append(
              schematicDef.Customization.SystemExtensions,
              schematic.SystemExtension{Name: overlay},
          )
      }
      
      schematicID, err := c.imageFactoryClient.SchematicCreate(ctx, schematicDef)
      if err != nil {
          return "", fmt.Errorf("failed to create schematic: %w", err)
      }
      
      return schematicID, nil
  }
  ```

#### Image URL Generation with Platform Support
- [ ] Implement `GetImageURL(ctx context.Context, schematicID, version, arch string) (string, error)` using platforms package:
  ```go
  func (c *Client) GetImageURL(ctx context.Context, schematicID, version, arch string) (string, error) {
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
  ```
- [ ] **Compression Decision**: Prefer `.raw.xz` (compressed) version:
  ```go
  func (c *Client) GetCompressedImageURL(ctx context.Context, schematicID, version, arch string) (string, error) {
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
  ```

### Phase 4: Cache Management
- [ ] Implement `CacheManager` struct
- [ ] Design cache key structure: `{talos_version}/{arch}/{schematic_id}/openstack.raw`
- [ ] Implement cache operations:
  - [ ] `Exists(key string) bool`
  - [ ] `Get(key string) (*os.File, error)`
  - [ ] `Set(key string, reader io.Reader) error`
  - [ ] `Delete(key string) error`
  - [ ] `Size() (int64, error)`
  - [ ] `Cleanup() error` - Remove old entries when cache exceeds limits
- [ ] Implement atomic cache writes using temporary files and rename
- [ ] Add cache directory initialization with proper permissions

### Phase 5: HTTP Handler Implementation
- [ ] Create `api/images/talos/handler.go`
- [ ] Implement handler struct following project patterns:
  ```go
  import (
      "log/slog"
      "net/http"
      "github.com/metal3-community/metal-boot/internal/config"
      "github.com/metal3-community/metal-boot/internal/talos"
      "github.com/siderolabs/image-factory/pkg/client"
  )

  type Handler struct {
      logger       *slog.Logger
      config       *config.TalosConfig
      client       *talos.Client
      cacheManager *talos.CacheManager
  }
  ```
- [ ] Implement `New(logger *slog.Logger, cfg *config.TalosConfig) http.Handler` constructor:
  ```go
  func New(logger *slog.Logger, cfg *config.TalosConfig) http.Handler {
      if !cfg.Enabled {
          return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
              http.Error(w, "Talos handler is disabled", http.StatusServiceUnavailable)
          })
      }

      client, err := talos.NewClient(cfg.BaseURL, logger)
      if err != nil {
          logger.Error("Failed to create Talos client", "error", err)
          return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
              http.Error(w, "Talos client initialization failed", http.StatusInternalServerError)
          })
      }

      cacheManager, err := talos.NewCacheManager(cfg.CacheDirectory, logger)
      if err != nil {
          logger.Error("Failed to create cache manager", "error", err)
          return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
              http.Error(w, "Cache initialization failed", http.StatusInternalServerError)
          })
      }

      return &Handler{
          logger:       logger,
          config:       cfg,
          client:       client,
          cacheManager: cacheManager,
      }
  }
  ```
- [ ] Implement `ServeHTTP(w http.ResponseWriter, r *http.Request)` with comprehensive request handling:
  ```go
  func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
      h.logger.Debug("Handling Talos image request", "path", r.URL.Path, "method", r.Method)

      if r.Method != http.MethodGet && r.Method != http.MethodHead {
          http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
          return
      }

      // Parse URL path: /images/talos/{version}/{arch}/openstack.raw
      pathParts, err := h.parseURLPath(r.URL.Path)
      if err != nil {
          h.logger.Warn("Invalid URL path", "path", r.URL.Path, "error", err)
          http.Error(w, "Invalid URL path", http.StatusBadRequest)
          return
      }

      // Parse query parameters
      extensions := h.parseCommaSeparated(r.URL.Query().Get("extensions"))
      overlays := h.parseCommaSeparated(r.URL.Query().Get("overlays"))

      // Process the request
      if err := h.handleImageRequest(w, r, pathParts, extensions, overlays); err != nil {
          h.logger.Error("Failed to handle image request", "error", err)
          http.Error(w, "Internal server error", http.StatusInternalServerError)
      }
  }
  ```
- [ ] Add URL path parsing and validation:
  ```go
  type RequestParams struct {
      Version string
      Arch    string
  }

  func (h *Handler) parseURLPath(path string) (*RequestParams, error) {
      // Remove prefix: /images/talos/
      path = strings.TrimPrefix(path, "/images/talos/")
      path = strings.TrimSuffix(path, "/openstack.raw")
      
      parts := strings.Split(path, "/")
      if len(parts) != 2 {
          return nil, fmt.Errorf("invalid path format, expected: /images/talos/{version}/{arch}/openstack.raw")
      }

      version := parts[0]
      arch := parts[1]

      // Validate architecture
      if arch != "amd64" && arch != "arm64" {
          return nil, fmt.Errorf("unsupported architecture: %s", arch)
      }

      return &RequestParams{
          Version: version,
          Arch:    arch,
      }, nil
  }
  ```
- [ ] Add query parameter parsing:
  ```go
  func (h *Handler) parseCommaSeparated(param string) []string {
      if param == "" {
          return nil
      }
      
      var result []string
      for _, item := range strings.Split(param, ",") {
          if trimmed := strings.TrimSpace(item); trimmed != "" {
              result = append(result, trimmed)
          }
      }
      return result
  }
  ```

### Phase 6: Stream-and-Cache Implementation
- [ ] Implement dual-writer pattern for simultaneous response streaming and caching:
  ```go
  import (
      "compress/gzip"
      "crypto/sha256"
      "io"
  )

  type StreamAndCacheWriter struct {
      responseWriter http.ResponseWriter
      cacheWriter    io.Writer
      hash           hash.Hash
      gzipReader     *gzip.Reader
      totalBytes     int64
  }

  func NewStreamAndCacheWriter(w http.ResponseWriter, cacheWriter io.Writer) *StreamAndCacheWriter {
      return &StreamAndCacheWriter{
          responseWriter: w,
          cacheWriter:    cacheWriter,
          hash:           sha256.New(),
      }
  }

  func (s *StreamAndCacheWriter) Write(p []byte) (n int, err error) {
      // Write to hash for checksum
      s.hash.Write(p)
      
      // Write to response
      n1, err1 := s.responseWriter.Write(p)
      
      // Write to cache
      n2, err2 := s.cacheWriter.Write(p)
      
      s.totalBytes += int64(n1)
      
      // Return the minimum bytes written and any error
      if err1 != nil {
          return n1, err1
      }
      if err2 != nil {
          return n2, err2
      }
      
      return min(n1, n2), nil
  }

  func (s *StreamAndCacheWriter) Checksum() string {
      return fmt.Sprintf("%x", s.hash.Sum(nil))
  }
  ```
- [ ] Implement the main request handling logic:
  ```go
  func (h *Handler) handleImageRequest(w http.ResponseWriter, r *http.Request, params *RequestParams, extensions, overlays []string) error {
      ctx := r.Context()
      
      // Resolve version ("latest" → actual version)
      actualVersion := params.Version
      if params.Version == "latest" {
          var err error
          actualVersion, err = h.client.GetLatestVersion(ctx)
          if err != nil {
              return fmt.Errorf("failed to resolve latest version: %w", err)
          }
      }
      
      // Validate extensions and overlays exist for this version
      if err := h.validateExtensionsAndOverlays(ctx, actualVersion, extensions, overlays); err != nil {
          return fmt.Errorf("validation failed: %w", err)
      }
      
      // Create schematic
      schematicID, err := h.client.CreateSchematic(ctx, extensions, overlays)
      if err != nil {
          return fmt.Errorf("failed to create schematic: %w", err)
      }
      
      // Generate cache key
      cacheKey := h.generateCacheKey(actualVersion, params.Arch, schematicID)
      
      // Check cache first
      if h.cacheManager.Exists(cacheKey) {
          return h.serveCachedImage(w, cacheKey)
      }
      
      // Stream and cache from upstream
      return h.streamAndCacheImage(w, r, actualVersion, params.Arch, schematicID, cacheKey)
  }
  ```
- [ ] Handle gzip decompression during streaming:
  ```go
  func (h *Handler) streamAndCacheImage(w http.ResponseWriter, r *http.Request, version, arch, schematicID, cacheKey string) error {
      ctx := r.Context()
      
      // Get compressed image URL
      imageURL, err := h.client.GetCompressedImageURL(ctx, schematicID, version, arch)
      if err != nil {
          return fmt.Errorf("failed to get image URL: %w", err)
      }
      
      // Download compressed image
      resp, err := http.Get(imageURL)
      if err != nil {
          return fmt.Errorf("failed to download image: %w", err)
      }
      defer resp.Body.Close()
      
      if resp.StatusCode != http.StatusOK {
          return fmt.Errorf("upstream returned status %d", resp.StatusCode)
      }
      
      // Create gzip reader for decompression
      gzipReader, err := gzip.NewReader(resp.Body)
      if err != nil {
          return fmt.Errorf("failed to create gzip reader: %w", err)
      }
      defer gzipReader.Close()
      
      // Set response headers
      w.Header().Set("Content-Type", "application/octet-stream")
      w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"talos-%s-%s-openstack.raw\"", version, arch))
      
      // Create cache file
      cacheFile, err := h.cacheManager.CreateCacheFile(cacheKey)
      if err != nil {
          return fmt.Errorf("failed to create cache file: %w", err)
      }
      defer cacheFile.Close()
      
      // Create dual writer
      dualWriter := NewStreamAndCacheWriter(w, cacheFile)
      
      // Stream decompressed data to both response and cache
      _, err = io.Copy(dualWriter, gzipReader)
      if err != nil {
          // Clean up partial cache file on error
          h.cacheManager.DeleteCacheFile(cacheKey)
          return fmt.Errorf("failed to stream image: %w", err)
      }
      
      // Finalize cache file
      if err := h.cacheManager.FinalizeCacheFile(cacheKey, dualWriter.Checksum()); err != nil {
          h.logger.Warn("Failed to finalize cache file", "key", cacheKey, "error", err)
      }
      
      h.logger.Info("Successfully streamed and cached image", 
          "version", version, 
          "arch", arch, 
          "schematic_id", schematicID,
          "bytes", dualWriter.totalBytes,
          "checksum", dualWriter.Checksum())
      
      return nil
  }
  ```
- [ ] Add validation for extensions and overlays:
  ```go
  func (h *Handler) validateExtensionsAndOverlays(ctx context.Context, version string, extensions, overlays []string) error {
      // Validate extensions
      if len(extensions) > 0 {
          availableExtensions, err := h.client.GetExtensionsVersions(ctx, version)
          if err != nil {
              return fmt.Errorf("failed to get available extensions: %w", err)
          }
          
          availableNames := make(map[string]bool)
          for _, ext := range availableExtensions {
              availableNames[ext.Name] = true
          }
          
          for _, ext := range extensions {
              if !availableNames[ext] {
                  return fmt.Errorf("extension %q is not available for version %s", ext, version)
              }
          }
      }
      
      // Validate overlays
      if len(overlays) > 0 {
          availableOverlays, err := h.client.GetOverlaysVersions(ctx, version)
          if err != nil {
              return fmt.Errorf("failed to get available overlays: %w", err)
          }
          
          availableNames := make(map[string]bool)
          for _, overlay := range availableOverlays {
              availableNames[overlay.Name] = true
          }
          
          for _, overlay := range overlays {
              if !availableNames[overlay] {
                  return fmt.Errorf("overlay %q is not available for version %s", overlay, version)
              }
          }
      }
      
      return nil
  }
  ```
- [ ] Implement cache key generation:
  ```go
  func (h *Handler) generateCacheKey(version, arch, schematicID string) string {
      return fmt.Sprintf("%s/%s/%s/openstack.raw", version, arch, schematicID)
  }
  ```

### Phase 7: Error Handling and Validation
- [ ] Add comprehensive input validation:
  - [ ] Talos version format validation
  - [ ] Architecture validation (amd64, arm64)
  - [ ] Extension/overlay name validation
- [ ] Implement graceful error responses:
  - [ ] 400 Bad Request for invalid parameters
  - [ ] 404 Not Found for unsupported versions/extensions
  - [ ] 500 Internal Server Error for upstream failures
  - [ ] 503 Service Unavailable for temporary failures
- [ ] Add structured logging with request context
- [ ] Implement request timeout handling
- [ ] Add retry logic for transient upstream failures

### Phase 8: Integration and Testing
- [ ] Add handler registration in `cmd/metal-boot/main.go`:
  ```go
  import "github.com/metal3-community/metal-boot/api/images/talos"
  
  // In the API server setup section:
  if cfg.Talos.Enabled {
      apiServer.AddHandler("/images/talos/", talos.New(slogger, &cfg.Talos))
  }
  ```
- [ ] Create comprehensive test suite:
  - [ ] Unit tests for URL parsing and validation:
    ```go
    func TestParseURLPath(t *testing.T) {
        tests := []struct {
            name     string
            path     string
            expected *RequestParams
            wantErr  bool
        }{
            {
                name: "valid path",
                path: "/images/talos/v1.11.1/amd64/openstack.raw",
                expected: &RequestParams{Version: "v1.11.1", Arch: "amd64"},
                wantErr: false,
            },
            {
                name: "latest version",
                path: "/images/talos/latest/arm64/openstack.raw", 
                expected: &RequestParams{Version: "latest", Arch: "arm64"},
                wantErr: false,
            },
            {
                name: "invalid architecture",
                path: "/images/talos/v1.11.1/x86/openstack.raw",
                wantErr: true,
            },
        }
        // Test implementation...
    }
    ```
  - [ ] Unit tests for schematic generation using the actual packages:
    ```go
    func TestCreateSchematic(t *testing.T) {
        client, err := talos.NewClient("https://factory.talos.dev", slog.Default())
        require.NoError(t, err)
        
        schematicID, err := client.CreateSchematic(context.Background(), 
            []string{"qemu-guest-agent"}, 
            []string{"rpi-generic"})
        require.NoError(t, err)
        assert.Len(t, schematicID, 64) // SHA256 hash length
    }
    ```
  - [ ] Integration tests with mock Talos factory API
  - [ ] Cache behavior tests
  - [ ] Stream-and-cache functionality tests
- [ ] Add OpenTelemetry tracing support following project patterns:
  ```go
  import "go.opentelemetry.io/otel"

  func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
      tracer := otel.Tracer("talos-handler")
      ctx, span := tracer.Start(r.Context(), "talos.ServeHTTP")
      defer span.End()
      
      // Add tracing attributes
      span.SetAttributes(
          attribute.String("talos.version", params.Version),
          attribute.String("talos.arch", params.Arch),
          attribute.StringSlice("talos.extensions", extensions),
          attribute.StringSlice("talos.overlays", overlays),
      )
      
      // Continue with request handling...
  }
  ```
- [ ] Test with various client scenarios (curl, automated tools)

### Phase 9: Documentation and Configuration
- [ ] Update `README.md` with Talos handler documentation
- [ ] Add configuration examples for common use cases
- [ ] Document supported extensions and overlays
- [ ] Add troubleshooting guide for common issues
- [ ] Create example API usage documentation

## Required Dependencies

### Go Modules to Add
```
github.com/blang/semver/v4                      // Semantic version parsing and comparison
github.com/siderolabs/image-factory/pkg/client // Talos image factory HTTP client
github.com/siderolabs/image-factory/pkg/schematic // Schematic definitions and structures
github.com/siderolabs/talos/pkg/machinery/platforms // Platform definitions (OpenStack, etc.)
```

### Standard Library Packages
```
compress/gzip     // Gzip decompression for .raw.xz files
crypto/sha256     // Checksum calculation  
encoding/json     // JSON parsing for API responses
net/http          // HTTP client
path/filepath     // Path manipulation
strings           // String processing
context           // Request context handling
fmt               // String formatting
io                // I/O operations
```

## Key Implementation Notes

### Leveraging Existing Talos Packages
The implementation uses three key packages from the Talos ecosystem:

1. **`github.com/siderolabs/image-factory/pkg/client`**: Provides HTTP client for the image factory API
   - `client.NewClient(baseURL)` - Creates HTTP client
   - `client.Versions(ctx)` - Gets available Talos versions  
   - `client.ExtensionsVersions(ctx, version)` - Gets extensions for version
   - `client.OverlaysVersions(ctx, version)` - Gets overlays for version
   - `client.SchematicCreate(ctx, schematic)` - Creates schematic and returns ID

2. **`github.com/siderolabs/image-factory/pkg/schematic`**: Provides schematic data structures
   - `schematic.Schematic` - Root schematic structure
   - `schematic.CustomizationSpec` - Customization specification
   - `schematic.SystemExtension` - Individual extension/overlay definition

3. **`github.com/siderolabs/talos/pkg/machinery/platforms`**: Provides platform definitions
   - `platforms.CloudPlatforms()` - Lists available cloud platforms
   - `Platform.DiskImageDefaultPath(arch)` - Gets correct image path for architecture
   - Used to generate proper OpenStack image URLs

### Schematic Generation Pattern
Extensions and overlays are both added as SystemExtensions in the schematic:
```yaml
customization:
  systemExtensions:
    - name: qemu-guest-agent       # extension
    - name: nvidia-open-gpu-kernel-modules  # extension  
    - name: rpi-generic            # overlay
```

### Error Handling Patterns
Follow the project's structured logging patterns:
```go
logger.Error("Failed to fetch Talos versions", "error", err, "url", url)
```

### Configuration Integration
Leverage Viper's automatic environment variable binding:
- `talos.enabled` → `TALOS_ENABLED`
- `talos.base_url` → `TALOS_BASE_URL`
- `talos.cache_directory` → `TALOS_CACHE_DIRECTORY`

### Cache Key Design
Use hierarchical cache keys that include all parameters:
```
cache/{talos_version}/{arch}/{schematic_hash}/openstack.raw
```

This ensures unique caching per parameter combination while maintaining predictable lookup paths.

### Stream Efficiency Decision
**Use compressed (.raw.xz) downloads** because:
1. Network bandwidth is typically the limiting factor
2. Gzip decompression is computationally lightweight
3. Reduces server load on Talos factory
4. Provides better user experience on slower connections
5. CPU cost of decompression is minimal compared to I/O wait times

### Platform-Aware URL Generation
Use the platforms package to generate correct URLs:
```go
// Get OpenStack platform definition
openstackPlatform := findPlatform("openstack", platforms.CloudPlatforms())

// Generate proper image path for architecture  
imagePath := openstackPlatform.DiskImageDefaultPath(arch)

// Result: "openstack-amd64.raw" or "openstack-arm64.raw"
```

## Security Considerations
- [ ] Validate all user inputs to prevent path traversal attacks
- [ ] Implement request rate limiting if needed
- [ ] Ensure cache directory permissions are restrictive (0755)
- [ ] Add request size limits to prevent DoS attacks
- [ ] Validate checksums of downloaded images when available
