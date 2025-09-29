package ironic

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/metal3-community/metal-boot/internal/util"
	toml "github.com/pelletier/go-toml/v2"
)

type Config struct {
	Default                    DefaultConfig                    `toml:"DEFAULT,omitempty"`
	Agent                      AgentConfig                      `toml:"agent,omitempty"`
	API                        APIConfig                        `toml:"api,omitempty"`
	Conductor                  ConductorConfig                  `toml:"conductor,omitempty"`
	Database                   DatabaseConfig                   `toml:"database,omitempty"`
	Deploy                     DeployConfig                     `toml:"deploy,omitempty"`
	DHCP                       DHCPConfig                       `toml:"dhcp,omitempty"`
	Inspector                  InspectorConfig                  `toml:"inspector,omitempty"`
	AutoDiscovery              AutoDiscoveryConfig              `toml:"auto_discovery,omitempty"`
	IPMI                       IPMIConfig                       `toml:"ipmi,omitempty"`
	JSONRPC                    JSONRPCConfig                    `toml:"json_rpc,omitempty"`
	Nova                       NovaConfig                       `toml:"nova,omitempty"`
	OsloMessagingNotifications OsloMessagingNotificationsConfig `toml:"oslo_messaging_notifications,omitempty"`
	SensorData                 SensorDataConfig                 `toml:"sensor_data,omitempty"`
	Metrics                    MetricsConfig                    `toml:"metrics,omitempty"`
	PXE                        PXEConfig                        `toml:"pxe,omitempty"`
	Redfish                    RedfishConfig                    `toml:"redfish,omitempty"`
	ILO                        ILOConfig                        `toml:"ilo,omitempty"`
	IRMC                       IRMCConfig                       `toml:"irmc,omitempty"`
	ServiceCatalog             ServiceCatalogConfig             `toml:"service_catalog,omitempty"`
	SSL                        SSLConfig                        `toml:"ssl,omitempty"`

	// ProcessManager configuration (not part of TOML config)
	SocketPath string `toml:"-"`
	ConfigPath string `toml:"-"`
	SharedRoot string `toml:"-"`
	SkipDBSync bool   `toml:"-"`
}

type DefaultConfig struct {
	AuthStrategy                string `toml:"auth_strategy,omitempty"`
	Debug                       *bool  `toml:"debug,omitempty"`
	DefaultDeployInterface      string `toml:"default_deploy_interface,omitempty"`
	DefaultInspectInterface     string `toml:"default_inspect_interface,omitempty"`
	DefaultNetworkInterface     string `toml:"default_network_interface,omitempty"`
	EnabledBiosInterfaces       string `toml:"enabled_bios_interfaces,omitempty"`
	EnabledBootInterfaces       string `toml:"enabled_boot_interfaces,omitempty"`
	EnabledDeployInterfaces     string `toml:"enabled_deploy_interfaces,omitempty"`
	EnabledFirmwareInterfaces   string `toml:"enabled_firmware_interfaces,omitempty"`
	EnabledHardwareTypes        string `toml:"enabled_hardware_types,omitempty"`
	EnabledInspectInterfaces    string `toml:"enabled_inspect_interfaces,omitempty"`
	EnabledManagementInterfaces string `toml:"enabled_management_interfaces,omitempty"`
	EnabledNetworkInterfaces    string `toml:"enabled_network_interfaces,omitempty"`
	EnabledPowerInterfaces      string `toml:"enabled_power_interfaces,omitempty"`
	EnabledRaidInterfaces       string `toml:"enabled_raid_interfaces,omitempty"`
	EnabledVendorInterfaces     string `toml:"enabled_vendor_interfaces,omitempty"`
	GrubConfigPath              string `toml:"grub_config_path,omitempty"`
	HashRingAlgorithm           string `toml:"hash_ring_algorithm,omitempty"`
	Host                        string `toml:"host,omitempty"`
	IsolinuxBin                 string `toml:"isolinux_bin,omitempty"`
	LogFile                     string `toml:"log_file,omitempty"`
	MyIP                        string `toml:"my_ip,omitempty"`
	RPCTransport                string `toml:"rpc_transport,omitempty"`
	UseRPCForDatabase           *bool  `toml:"use_rpc_for_database,omitempty"`
	UseStderr                   *bool  `toml:"use_stderr,omitempty"`
	WebserverVerifyCA           *bool  `toml:"webserver_verify_ca,omitempty"`
}

type AgentConfig struct {
	DeployLogsCollect   string `toml:"deploy_logs_collect,omitempty"`
	DeployLogsLocalPath string `toml:"deploy_logs_local_path,omitempty"`
	MaxCommandAttempts  int    `toml:"max_command_attempts,omitempty"`
	CertificatesPath    string `toml:"certificates_path,omitempty"`
	ImageDownloadSource string `toml:"image_download_source,omitempty"`
}

type APIConfig struct {
	UnixSocket     string `toml:"unix_socket,omitempty"`
	UnixSocketMode string `toml:"unix_socket_mode,omitempty"`
	PublicEndpoint string `toml:"public_endpoint,omitempty"`
	HostIP         string `toml:"host_ip,omitempty"`
	Port           int    `toml:"port,omitempty"`
	EnableSSLAPI   *bool  `toml:"enable_ssl_api,omitempty"`
	APIWorkers     int    `toml:"api_workers,omitempty"`
}

type ArchMap map[string]string

func (a *ArchMap) MarshalText() ([]byte, error) {
	if a == nil {
		return []byte{}, nil
	}
	am := *a
	if len(am) == 0 {
		return []byte{}, nil
	}
	entries := []string{}
	for k, v := range am {
		entries = append(entries, fmt.Sprintf("%s:%s", k, v))
	}
	return []byte(strings.Join(entries, ",")), nil
}

func (a *ArchMap) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}
	am := ArchMap{}
	ts := strings.TrimPrefix(string(text), "{")
	ts = strings.TrimSuffix(ts, "}")
	for _, entry := range strings.Split(ts, ",") {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid ArchMap entry: %s", entry)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" || value == "" {
			return fmt.Errorf("invalid ArchMap entry: %s", entry)
		}
		am[key] = value
	}
	*a = am
	return nil
}

type ConductorConfig struct {
	AutomatedClean             *bool   `toml:"automated_clean,omitempty"`
	APIURL                     string  `toml:"api_url,omitempty"`
	DeployCallbackTimeout      int     `toml:"deploy_callback_timeout,omitempty"`
	BootloaderByArch           ArchMap `toml:"bootloader_by_arch,omitempty"`
	VerifyStepPriorityOverride string  `toml:"verify_step_priority_override,omitempty"`
	NodeHistory                *bool   `toml:"node_history,omitempty"`
	PowerStateChangeTimeout    int     `toml:"power_state_change_timeout,omitempty"`
	DeployKernel               string  `toml:"deploy_kernel,omitempty"`
	DeployKernelByArch         ArchMap `toml:"deploy_kernel_by_arch,omitempty"`
	DeployRamdisk              string  `toml:"deploy_ramdisk,omitempty"`
	DeployRamdiskByArch        ArchMap `toml:"deploy_ramdisk_by_arch,omitempty"`
	DisableDeepImageInspection *bool   `toml:"disable_deep_image_inspection,omitempty"`
	FileURLAllowedPaths        string  `toml:"file_url_allowed_paths,omitempty"`
}

type DatabaseConfig struct {
	Connection        string `toml:"connection,omitempty"`
	SqliteSynchronous *bool  `toml:"sqlite_synchronous,omitempty"`
}

type DeployConfig struct {
	DefaultBootOption            string `toml:"default_boot_option,omitempty"`
	EraseDevicesMetadataPriority int    `toml:"erase_devices_metadata_priority,omitempty"`
	EraseDevicesPriority         int    `toml:"erase_devices_priority,omitempty"`
	HTTPRoot                     string `toml:"http_root,omitempty"`
	HTTPURL                      string `toml:"http_url,omitempty"`
	FastTrack                    *bool  `toml:"fast_track,omitempty"`
	RamdiskImageDownloadSource   string `toml:"ramdisk_image_download_source,omitempty"`
	ExternalHTTPURL              string `toml:"external_http_url,omitempty"`
	ExternalCallbackURL          string `toml:"external_callback_url,omitempty"`
}

type DHCPConfig struct {
	DHCPProvider string `toml:"dhcp_provider,omitempty"`
}

type InspectorConfig struct {
	RequireManagedBoot *bool  `toml:"require_managed_boot,omitempty"`
	PowerOff           string `toml:"power_off,omitempty"`
	ExtraKernelParams  string `toml:"extra_kernel_params,omitempty"`
	Hooks              string `toml:"hooks,omitempty"`
	AddPorts           string `toml:"add_ports,omitempty"`
	KeepPorts          string `toml:"keep_ports,omitempty"`
}

type AutoDiscoveryConfig struct {
	Enabled string `toml:"enabled,omitempty"`
	Driver  string `toml:"driver,omitempty"`
}

type IPMIConfig struct {
	UseIpmitoolRetries  *bool  `toml:"use_ipmitool_retries,omitempty"`
	MinCommandInterval  int    `toml:"min_command_interval,omitempty"`
	CommandRetryTimeout int    `toml:"command_retry_timeout,omitempty"`
	CipherSuiteVersions string `toml:"cipher_suite_versions,omitempty"`
}

type JSONRPCConfig struct {
	AuthStrategy   string `toml:"auth_strategy,omitempty"`
	HostIP         string `toml:"host_ip,omitempty"`
	UnixSocket     string `toml:"unix_socket,omitempty"`
	UnixSocketMode string `toml:"unix_socket_mode,omitempty"`
	Port           int    `toml:"port,omitempty"`

	Enabled bool `toml:"-"`
}

type NovaConfig struct {
	SendPowerNotifications *bool `toml:"send_power_notifications,omitempty"`
}

type OsloMessagingNotificationsConfig struct {
	Driver       string `toml:"driver,omitempty"`
	Location     string `toml:"location,omitempty"`
	TransportURL string `toml:"transport_url,omitempty"`
}

type SensorDataConfig struct {
	SendSensorData *bool `toml:"send_sensor_data,omitempty"`
	Interval       int   `toml:"interval,omitempty"`
}

type MetricsConfig struct {
	Backend string `toml:"backend,omitempty"`
}

type PXEConfig struct {
	BootRetryTimeout      int    `toml:"boot_retry_timeout,omitempty"`
	ImagesPath            string `toml:"images_path,omitempty"`
	InstanceMasterPath    string `toml:"instance_master_path,omitempty"`
	TFTPMasterPath        string `toml:"tftp_master_path,omitempty"`
	TFTPRoot              string `toml:"tftp_root,omitempty"`
	KernelAppendParams    string `toml:"kernel_append_params,omitempty"`
	EnableNetbootFallback *bool  `toml:"enable_netboot_fallback,omitempty"`
	IPXEFallbackScript    string `toml:"ipxe_fallback_script,omitempty"`
	IPXEConfigTemplate    string `toml:"ipxe_config_template,omitempty"`
}

type RedfishConfig struct {
	UseSwift           *bool  `toml:"use_swift,omitempty"`
	KernelAppendParams string `toml:"kernel_append_params,omitempty"`
}

type ILOConfig struct {
	KernelAppendParams    string `toml:"kernel_append_params,omitempty"`
	UseWebServerForImages *bool  `toml:"use_web_server_for_images,omitempty"`
}

type IRMCConfig struct {
	KernelAppendParams string `toml:"kernel_append_params,omitempty"`
}

type ServiceCatalogConfig struct {
	EndpointOverride string `toml:"endpoint_override,omitempty"`
}

type SSLConfig struct {
	CertFile string `toml:"cert_file,omitempty"`
	KeyFile  string `toml:"key_file,omitempty"`
}

// SetDefaults populates the configuration with sensible defaults for all fields.
// This should be called before applying any user-provided configuration values.
func (c *Config) SetDefaults() {
	// Default section
	if c.Default.AuthStrategy == "" {
		c.Default.AuthStrategy = "noauth"
	}
	if c.Default.Debug == nil {
		c.Default.Debug = util.Ptr(true)
	}
	if c.Default.DefaultDeployInterface == "" {
		c.Default.DefaultDeployInterface = "direct"
	}
	if c.Default.DefaultInspectInterface == "" {
		c.Default.DefaultInspectInterface = "agent"
	}
	if c.Default.DefaultNetworkInterface == "" {
		c.Default.DefaultNetworkInterface = "noop"
	}
	if c.Default.EnabledBiosInterfaces == "" {
		c.Default.EnabledBiosInterfaces = "no-bios,redfish,idrac-redfish,ilo"
	}
	if c.Default.EnabledBootInterfaces == "" {
		c.Default.EnabledBootInterfaces = "ipxe,ilo-ipxe,pxe,ilo-pxe,fake,redfish-virtual-media,idrac-redfish-virtual-media,ilo-virtual-media,redfish-https"
	}
	if c.Default.EnabledDeployInterfaces == "" {
		c.Default.EnabledDeployInterfaces = "direct,fake,ramdisk,custom-agent"
	}
	if c.Default.EnabledFirmwareInterfaces == "" {
		c.Default.EnabledFirmwareInterfaces = "no-firmware,fake,redfish"
	}
	if c.Default.EnabledHardwareTypes == "" {
		c.Default.EnabledHardwareTypes = "ipmi,idrac,fake-hardware,redfish,manual-management,ilo,ilo5"
	}
	if c.Default.EnabledInspectInterfaces == "" {
		c.Default.EnabledInspectInterfaces = "agent,fake,redfish,ilo"
	}
	if c.Default.EnabledManagementInterfaces == "" {
		c.Default.EnabledManagementInterfaces = "ipmitool,fake,redfish,idrac-redfish,ilo,ilo5,noop"
	}
	if c.Default.EnabledNetworkInterfaces == "" {
		c.Default.EnabledNetworkInterfaces = "noop"
	}
	if c.Default.EnabledPowerInterfaces == "" {
		c.Default.EnabledPowerInterfaces = "ipmitool,fake,redfish,idrac-redfish,ilo"
	}
	if c.Default.EnabledRaidInterfaces == "" {
		c.Default.EnabledRaidInterfaces = "no-raid,agent,fake,redfish,idrac-redfish,ilo5"
	}
	if c.Default.EnabledVendorInterfaces == "" {
		c.Default.EnabledVendorInterfaces = "no-vendor,ipmitool,idrac-redfish,redfish,ilo,fake"
	}
	if c.Default.RPCTransport == "" {
		c.Default.RPCTransport = "none"
	}
	if c.Default.UseStderr == nil {
		c.Default.UseStderr = util.Ptr(true)
	}
	if c.Default.HashRingAlgorithm == "" {
		c.Default.HashRingAlgorithm = "sha256"
	}
	if c.Default.MyIP == "" {
		c.Default.MyIP = "127.0.0.1"
	}
	if c.Default.Host == "" {
		c.Default.Host = "localhost"
	}

	// Agent section
	if c.Agent.DeployLogsCollect == "" {
		c.Agent.DeployLogsCollect = "always"
	}
	if c.Agent.MaxCommandAttempts == 0 {
		c.Agent.MaxCommandAttempts = 30
	}
	if c.Agent.ImageDownloadSource == "" {
		c.Agent.ImageDownloadSource = "http"
	}

	// API section
	if c.API.HostIP == "" {
		c.API.HostIP = "127.0.0.1"
	}
	if c.API.EnableSSLAPI == nil {
		c.API.EnableSSLAPI = util.Ptr(false)
	}
	if c.API.APIWorkers == 0 {
		c.API.APIWorkers = 0
	}

	// Conductor section
	if c.Conductor.AutomatedClean == nil {
		c.Conductor.AutomatedClean = util.Ptr(false)
	}
	if c.Conductor.DeployCallbackTimeout == 0 {
		c.Conductor.DeployCallbackTimeout = 4800
	}
	if c.Conductor.VerifyStepPriorityOverride == "" {
		c.Conductor.VerifyStepPriorityOverride = "management.clear_job_queue:90"
	}
	if c.Conductor.NodeHistory == nil {
		c.Conductor.NodeHistory = util.Ptr(false)
	}
	if c.Conductor.PowerStateChangeTimeout == 0 {
		c.Conductor.PowerStateChangeTimeout = 120
	}
	if c.Conductor.DisableDeepImageInspection == nil {
		c.Conductor.DisableDeepImageInspection = util.Ptr(true)
	}

	// Deploy section
	if c.Deploy.DefaultBootOption == "" {
		c.Deploy.DefaultBootOption = "local"
	}
	if c.Deploy.EraseDevicesMetadataPriority == 0 {
		c.Deploy.EraseDevicesMetadataPriority = 10
	}
	if c.Deploy.EraseDevicesPriority == 0 {
		c.Deploy.EraseDevicesPriority = 0
	}
	if c.Deploy.FastTrack == nil {
		c.Deploy.FastTrack = util.Ptr(false)
	}

	// DHCP section
	if c.DHCP.DHCPProvider == "" {
		c.DHCP.DHCPProvider = "none"
	}

	// Inspector section
	if c.Inspector.RequireManagedBoot == nil {
		c.Inspector.RequireManagedBoot = util.Ptr(false)
	}
	if c.Inspector.PowerOff == "" {
		c.Inspector.PowerOff = "true"
	}
	if c.Inspector.ExtraKernelParams == "" {
		c.Inspector.ExtraKernelParams = "ipa-inspection-collectors=default ipa-enable-vlan-interfaces=all ipa-inspection-dhcp-all-interfaces=1 ipa-collect-lldp=1"
	}
	if c.Inspector.Hooks == "" {
		c.Inspector.Hooks = "$default_hooks,parse-lldp"
	}
	if c.Inspector.AddPorts == "" {
		c.Inspector.AddPorts = "all"
	}
	if c.Inspector.KeepPorts == "" {
		c.Inspector.KeepPorts = "present"
	}

	// AutoDiscovery section
	if c.AutoDiscovery.Enabled == "" {
		c.AutoDiscovery.Enabled = "false"
	}
	if c.AutoDiscovery.Driver == "" {
		c.AutoDiscovery.Driver = "ipmi"
	}

	// IPMI section
	if c.IPMI.UseIpmitoolRetries == nil {
		c.IPMI.UseIpmitoolRetries = util.Ptr(false)
	}
	if c.IPMI.MinCommandInterval == 0 {
		c.IPMI.MinCommandInterval = 5
	}
	if c.IPMI.CommandRetryTimeout == 0 {
		c.IPMI.CommandRetryTimeout = 60
	}
	if c.IPMI.CipherSuiteVersions == "" {
		c.IPMI.CipherSuiteVersions = "3,17"
	}

	// JSONRPC section
	if c.JSONRPC.AuthStrategy == "" {
		c.JSONRPC.AuthStrategy = "noauth"
	}
	if c.JSONRPC.HostIP == "" {
		c.JSONRPC.HostIP = "127.0.0.1"
	}

	// Nova section
	if c.Nova.SendPowerNotifications == nil {
		c.Nova.SendPowerNotifications = util.Ptr(false)
	}

	// OsloMessagingNotifications section
	if c.OsloMessagingNotifications.Driver == "" {
		c.OsloMessagingNotifications.Driver = "noop"
	}
	if c.OsloMessagingNotifications.TransportURL == "" {
		c.OsloMessagingNotifications.TransportURL = "fake://"
	}

	// SensorData section
	if c.SensorData.SendSensorData == nil {
		c.SensorData.SendSensorData = util.Ptr(false)
	}
	if c.SensorData.Interval == 0 {
		c.SensorData.Interval = 160
	}

	// Metrics section
	if c.Metrics.Backend == "" {
		c.Metrics.Backend = "collector"
	}

	// PXE section
	if c.PXE.BootRetryTimeout == 0 {
		c.PXE.BootRetryTimeout = 1200
	}
	if c.PXE.KernelAppendParams == "" {
		c.PXE.KernelAppendParams = "nofb nomodeset vga=normal ipa-insecure=1 fips=1 systemd.journald.forward_to_console=yes"
	}
	if c.PXE.EnableNetbootFallback == nil {
		c.PXE.EnableNetbootFallback = util.Ptr(true)
	}
	if c.PXE.IPXEFallbackScript == "" {
		c.PXE.IPXEFallbackScript = "inspector.ipxe"
	}
	// if c.PXE.IPXEConfigTemplate == "" {
	// 	c.PXE.IPXEConfigTemplate = "/templates/ipxe_config.template"
	// }

	// Redfish section
	if c.Redfish.UseSwift == nil {
		c.Redfish.UseSwift = util.Ptr(false)
	}
	if c.Redfish.KernelAppendParams == "" {
		c.Redfish.KernelAppendParams = "nofb nomodeset vga=normal ipa-insecure=1 fips=1 systemd.journald.forward_to_console=yes"
	}

	// ILO section
	if c.ILO.KernelAppendParams == "" {
		c.ILO.KernelAppendParams = "nofb nomodeset vga=normal ipa-insecure=1 fips=1 systemd.journald.forward_to_console=yes"
	}
	if c.ILO.UseWebServerForImages == nil {
		c.ILO.UseWebServerForImages = util.Ptr(true)
	}

	// IRMC section
	if c.IRMC.KernelAppendParams == "" {
		c.IRMC.KernelAppendParams = "nofb nomodeset vga=normal ipa-insecure=1 fips=1 systemd.journald.forward_to_console=yes"
	}

	// ProcessManager-specific defaults for Unix socket operation
	if c.API.UnixSocketMode == "" {
		c.API.UnixSocketMode = "0666"
	}
	if c.JSONRPC.UnixSocketMode == "" {
		c.JSONRPC.UnixSocketMode = "0666"
	}
	if c.Database.Connection == "" {
		c.Database.Connection = "sqlite:////var/lib/ironic/ironic.db"
	}
}

// SetRuntimePaths configures paths that depend on ProcessManager configuration.
// This should be called after SetDefaults and after the ProcessManager paths are set.
func (c *Config) SetRuntimePaths(socketPath, sharedRoot string) {
	// Set runtime-specific paths that depend on ProcessManager configuration
	if c.Default.LogFile == "" {
		c.Default.LogFile = sharedRoot + "/log/ironic/ironic.log"
	}
	if c.API.UnixSocket == "" {
		c.API.UnixSocket = socketPath
	}
	if c.JSONRPC.UnixSocket == "" && c.JSONRPC.Enabled {
		c.JSONRPC.UnixSocket = "/tmp/ironic-rpc.sock"
	} else {
		c.Default.UseRPCForDatabase = util.Ptr(true)
	}
	if c.Conductor.APIURL == "" {
		c.Conductor.APIURL = "http+unix://" + socketPath
	}

	// Set shared root dependent paths
	if c.Agent.DeployLogsLocalPath == "" {
		c.Agent.DeployLogsLocalPath = sharedRoot + "/log/ironic/deploy"
	}
	if c.Deploy.HTTPRoot == "" {
		c.Deploy.HTTPRoot = filepath.Join(sharedRoot, "html")
	}
	if c.Conductor.FileURLAllowedPaths == "" {
		c.Conductor.FileURLAllowedPaths = strings.Join([]string{
			c.Deploy.HTTPRoot,
			"/templates",
		}, ",")
	}
	if c.OsloMessagingNotifications.Location == "" {
		c.OsloMessagingNotifications.Location = sharedRoot + "/ironic_prometheus_exporter"
	}
	if c.PXE.ImagesPath == "" {
		c.PXE.ImagesPath = filepath.Join(c.Deploy.HTTPRoot, "tmp")
	}
	if c.PXE.InstanceMasterPath == "" {
		c.PXE.InstanceMasterPath = filepath.Join(c.Deploy.HTTPRoot, "master_images")
	}
	archList := []string{"x86_64", "aarch64"}
	archLookup := map[string]string{
		"x86_64":  "amd64",
		"aarch64": "arm64",
	}
	bootloaderLookup := map[string]string{
		"x86_64":  "ipxe.efi",
		"aarch64": "snp.efi",
	}
	// Set architecture-specific defaults for bootloader, kernel, and ramdisk
	if len(c.Conductor.BootloaderByArch) == 0 {
		c.Conductor.BootloaderByArch = ArchMap{}
		for _, arch := range archList {
			bootloaderFile, ok := bootloaderLookup[arch]
			if !ok {
				bootloaderFile = "ipxe.efi"
			}
			bootloaderPath, err := url.JoinPath(c.Deploy.ExternalHTTPURL, bootloaderFile)
			if err == nil {
				c.Conductor.BootloaderByArch[arch] = bootloaderPath
			}
		}
	}
	if len(c.Conductor.DeployKernelByArch) == 0 {
		c.Conductor.DeployKernelByArch = ArchMap{}
		for _, arch := range archList {
			archPath, ok := archLookup[arch]
			if !ok {
				archPath = arch
			}
			fileUrl, err := url.JoinPath(
				"file://",
				filepath.Join(c.Deploy.HTTPRoot, "images", archPath, "ironic-python-agent.kernel"),
			)
			if err == nil {
				c.Conductor.DeployKernelByArch[arch] = fileUrl
			}
		}
	}
	if len(c.Conductor.DeployRamdiskByArch) == 0 {
		c.Conductor.DeployRamdiskByArch = ArchMap{}
		for _, arch := range archList {
			archPath, ok := archLookup[arch]
			if !ok {
				archPath = arch
			}
			fileUrl, err := url.JoinPath(
				"file://",
				filepath.Join(
					c.Deploy.HTTPRoot,
					"images",
					archPath,
					"ironic-python-agent.initramfs",
				),
			)
			if err == nil {
				c.Conductor.DeployRamdiskByArch[arch] = fileUrl
			}
		}
	}

	if c.PXE.TFTPRoot == "" {
		c.PXE.TFTPRoot = filepath.Join(sharedRoot, "tftpboot")
	}
	if c.PXE.TFTPMasterPath == "" {
		c.PXE.TFTPMasterPath = filepath.Join(c.PXE.TFTPRoot, "master_images")
	}
}

func (c *Config) Unmarshal(data []byte) error {
	return toml.Unmarshal(data, c)
}

func (c *Config) Marshal() ([]byte, error) {
	return toml.Marshal(c)
}
