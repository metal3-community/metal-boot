package ironic

import (
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
	UseStderr                   *bool  `toml:"use_stderr,omitempty"`
	WebserverVerifyCA           *bool  `toml:"webserver_verify_ca,omitempty"`
}

type AgentConfig struct {
	DeployLogsCollect   string `toml:"deploy_logs_collect,omitempty"`
	DeployLogsLocalPath string `toml:"deploy_logs_local_path,omitempty"`
	MaxCommandAttempts  int    `toml:"max_command_attempts,omitempty"`
	CertificatesPath    string `toml:"certificates_path,omitempty"`
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

type ConductorConfig struct {
	AutomatedClean             *bool  `toml:"automated_clean,omitempty"`
	APIURL                     string `toml:"api_url,omitempty"`
	DeployCallbackTimeout      int    `toml:"deploy_callback_timeout,omitempty"`
	BootloaderByArch           string `toml:"bootloader_by_arch,omitempty"`
	VerifyStepPriorityOverride string `toml:"verify_step_priority_override,omitempty"`
	NodeHistory                *bool  `toml:"node_history,omitempty"`
	PowerStateChangeTimeout    int    `toml:"power_state_change_timeout,omitempty"`
	DeployKernel               string `toml:"deploy_kernel,omitempty"`
	DeployKernelByArch         string `toml:"deploy_kernel_by_arch,omitempty"`
	DeployRamdisk              string `toml:"deploy_ramdisk,omitempty"`
	DeployRamdiskByArch        string `toml:"deploy_ramdisk_by_arch,omitempty"`
	DisableDeepImageInspection *bool  `toml:"disable_deep_image_inspection,omitempty"`
	FileURLAllowedPaths        string `toml:"file_url_allowed_paths,omitempty"`
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
	AuthStrategy string `toml:"auth_strategy,omitempty"`
	HostIP       string `toml:"host_ip,omitempty"`
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

func (c *Config) Unmarshal(data []byte) error {
	return toml.Unmarshal(data, c)
}

func (c *Config) Marshal() ([]byte, error) {
	return toml.Marshal(c)
}
