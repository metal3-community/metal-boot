package ironic

import (
	"testing"
)

func TestConfig_Unmarshal(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    Config
		wantErr bool
	}{
		{
			name: "valid TOML with DEFAULT section",
			data: []byte(`
[DEFAULT]
auth_strategy = "keystone"
debug = true
enabled_hardware_types = "ipmi,redfish"
`),
			want: Config{
				Default: DefaultConfig{
					AuthStrategy:         "keystone",
					Debug:                boolPtr(true),
					EnabledHardwareTypes: "ipmi,redfish",
				},
			},
			wantErr: false,
		},
		{
			name: "valid TOML with multiple sections",
			data: []byte(`
[api]
host_ip = "0.0.0.0"
port = 6385
enable_ssl_api = false

[database]
connection = "sqlite:///ironic.db"
sqlite_synchronous = true

[pxe]
boot_retry_timeout = 300
tftp_root = "/tftpboot"
`),
			want: Config{
				API: APIConfig{
					HostIP:       "0.0.0.0",
					Port:         6385,
					EnableSSLAPI: boolPtr(false),
				},
				Database: DatabaseConfig{
					Connection:        "sqlite:///ironic.db",
					SqliteSynchronous: boolPtr(true),
				},
				PXE: PXEConfig{
					BootRetryTimeout: 300,
					TFTPRoot:         "/tftpboot",
				},
			},
			wantErr: false,
		},
		{
			name:    "empty TOML",
			data:    []byte(``),
			want:    Config{},
			wantErr: false,
		},
		{
			name:    "invalid TOML",
			data:    []byte(`[invalid toml`),
			want:    Config{},
			wantErr: true,
		},
		{
			name: "comprehensive Ironic config",
			data: []byte(`
[DEFAULT]
auth_strategy = "noauth"
debug = true
default_deploy_interface = "direct"
default_inspect_interface = "agent"
default_network_interface = "noop"
enabled_bios_interfaces = "no-bios,redfish,idrac-redfish,ilo"
enabled_boot_interfaces = "ipxe,ilo-ipxe,pxe,ilo-pxe,fake,redfish-virtual-media,idrac-redfish-virtual-media,ilo-virtual-media,redfish-https"
enabled_deploy_interfaces = "direct,fake,ramdisk,custom-agent"
enabled_firmware_interfaces = "no-firmware,fake,redfish"
enabled_hardware_types = "ipmi,idrac,fake-hardware,redfish,manual-management,ilo,ilo5"
enabled_inspect_interfaces = "agent,fake,redfish,ilo"
enabled_management_interfaces = "ipmitool,fake,redfish,idrac-redfish,ilo,ilo5,noop"
enabled_network_interfaces = "noop"
enabled_power_interfaces = "ipmitool,fake,redfish,idrac-redfish,ilo"
enabled_raid_interfaces = "no-raid,agent,fake,redfish,idrac-redfish,ilo5"
enabled_vendor_interfaces = "no-vendor,ipmitool,idrac-redfish,redfish,ilo,fake"
rpc_transport = "json-rpc"
use_stderr = true
hash_ring_algorithm = "sha256"
my_ip = "0.0.0.0"
host = "ironic"
webserver_verify_ca = false
isolinux_bin = "/usr/share/syslinux/isolinux.bin"
grub_config_path = "EFI/centos/grub.cfg"

[agent]
deploy_logs_collect = "always"
deploy_logs_local_path = "/shared/log/ironic/deploy"
max_command_attempts = 30
certificates_path = ""

[api]
unix_socket = "/shared/ironic.sock"
unix_socket_mode = "0666"
public_endpoint = ""
host_ip = "::"
port = 6385
enable_ssl_api = false
api_workers = 0

[conductor]
automated_clean = false
deploy_callback_timeout = 4800
bootloader_by_arch = ""
verify_step_priority_override = "management.clear_job_queue:90"
node_history = false
power_state_change_timeout = 120
deploy_kernel = ""
deploy_kernel_by_arch = ""
deploy_ramdisk = ""
deploy_ramdisk_by_arch = ""
disable_deep_image_inspection = true
file_url_allowed_paths = "/shared/html/images,/templates"

[database]
connection = ""
sqlite_synchronous = false

[deploy]
default_boot_option = "local"
erase_devices_metadata_priority = 10
erase_devices_priority = 0
http_root = "/shared/html/"
http_url = "http://ironic:8080/"
fast_track = false
ramdisk_image_download_source = ""
external_http_url = "https://ironic:6385/v1/drivers/pxe/httpboot"
external_callback_url = ""

[dhcp]
dhcp_provider = "none"

[inspector]
require_managed_boot = false
power_off = "true"
extra_kernel_params = "ipa-inspection-collectors=default ipa-enable-vlan-interfaces=all ipa-inspection-dhcp-all-interfaces=1 ipa-collect-lldp=1"
hooks = "$default_hooks,parse-lldp"
add_ports = "all"
keep_ports = "present"

[auto_discovery]
enabled = "false"
driver = "ipmi"

[ipmi]
use_ipmitool_retries = false
min_command_interval = 5
command_retry_timeout = 60
cipher_suite_versions = "3,17"

[json_rpc]
auth_strategy = "noauth"
host_ip = "::"

[nova]
send_power_notifications = false

[oslo_messaging_notifications]
driver = "prometheus_exporter"
location = "/shared/ironic_prometheus_exporter"
transport_url = "fake://"

[sensor_data]
send_sensor_data = false
interval = 160

[metrics]
backend = "collector"

[pxe]
boot_retry_timeout = 1200
images_path = "/shared/html/tmp"
instance_master_path = "/shared/html/master_images"
tftp_master_path = "/shared/tftpboot/master_images"
tftp_root = "/shared/tftpboot"
kernel_append_params = "nofb nomodeset vga=normal ipa-insecure=1 fips=1 sshkey=\"\" systemd.journald.forward_to_console=yes"
enable_netboot_fallback = true
ipxe_fallback_script = "inspector.ipxe"
ipxe_config_template = "/templates/ipxe_config.template"

[redfish]
use_swift = false
kernel_append_params = "nofb nomodeset vga=normal ipa-insecure=1 fips=1 sshkey=\"\" systemd.journald.forward_to_console=yes"

[ilo]
kernel_append_params = "nofb nomodeset vga=normal ipa-insecure=1 fips=1 sshkey=\"\" systemd.journald.forward_to_console=yes"
use_web_server_for_images = true

[irmc]
kernel_append_params = "nofb nomodeset vga=normal ipa-insecure=1 fips=1 sshkey=\"\" systemd.journald.forward_to_console=yes"

[service_catalog]
endpoint_override = "http://ironic:6385/v1/"

[ssl]
cert_file = ""
key_file = ""
`),
			want: Config{
				Default: DefaultConfig{
					AuthStrategy:                "noauth",
					Debug:                       boolPtr(true),
					DefaultDeployInterface:      "direct",
					DefaultInspectInterface:     "agent",
					DefaultNetworkInterface:     "noop",
					EnabledBiosInterfaces:       "no-bios,redfish,idrac-redfish,ilo",
					EnabledBootInterfaces:       "ipxe,ilo-ipxe,pxe,ilo-pxe,fake,redfish-virtual-media,idrac-redfish-virtual-media,ilo-virtual-media,redfish-https",
					EnabledDeployInterfaces:     "direct,fake,ramdisk,custom-agent",
					EnabledFirmwareInterfaces:   "no-firmware,fake,redfish",
					EnabledHardwareTypes:        "ipmi,idrac,fake-hardware,redfish,manual-management,ilo,ilo5",
					EnabledInspectInterfaces:    "agent,fake,redfish,ilo",
					EnabledManagementInterfaces: "ipmitool,fake,redfish,idrac-redfish,ilo,ilo5,noop",
					EnabledNetworkInterfaces:    "noop",
					EnabledPowerInterfaces:      "ipmitool,fake,redfish,idrac-redfish,ilo",
					EnabledRaidInterfaces:       "no-raid,agent,fake,redfish,idrac-redfish,ilo5",
					EnabledVendorInterfaces:     "no-vendor,ipmitool,idrac-redfish,redfish,ilo,fake",
					RPCTransport:                "json-rpc",
					UseStderr:                   boolPtr(true),
					HashRingAlgorithm:           "sha256",
					MyIP:                        "0.0.0.0",
					Host:                        "ironic",
					WebserverVerifyCA:           boolPtr(false),
					IsolinuxBin:                 "/usr/share/syslinux/isolinux.bin",
					GrubConfigPath:              "EFI/centos/grub.cfg",
				},
				Agent: AgentConfig{
					DeployLogsCollect:   "always",
					DeployLogsLocalPath: "/shared/log/ironic/deploy",
					MaxCommandAttempts:  30,
					CertificatesPath:    "",
				},
				API: APIConfig{
					UnixSocket:     "/shared/ironic.sock",
					UnixSocketMode: "0666",
					PublicEndpoint: "",
					HostIP:         "::",
					Port:           6385,
					EnableSSLAPI:   boolPtr(false),
					APIWorkers:     0,
				},
				Conductor: ConductorConfig{
					AutomatedClean:             boolPtr(false),
					DeployCallbackTimeout:      4800,
					BootloaderByArch:           "",
					VerifyStepPriorityOverride: "management.clear_job_queue:90",
					NodeHistory:                boolPtr(false),
					PowerStateChangeTimeout:    120,
					DeployKernel:               "",
					DeployKernelByArch:         "",
					DeployRamdisk:              "",
					DeployRamdiskByArch:        "",
					DisableDeepImageInspection: boolPtr(true),
					FileURLAllowedPaths:        "/shared/html/images,/templates",
				},
				Database: DatabaseConfig{
					Connection:        "",
					SqliteSynchronous: boolPtr(false),
				},
				Deploy: DeployConfig{
					DefaultBootOption:            "local",
					EraseDevicesMetadataPriority: 10,
					EraseDevicesPriority:         0,
					HTTPRoot:                     "/shared/html/",
					HTTPURL:                      "http://ironic:8080/",
					FastTrack:                    boolPtr(false),
					RamdiskImageDownloadSource:   "",
					ExternalHTTPURL:              "https://ironic:6385/v1/drivers/pxe/httpboot",
					ExternalCallbackURL:          "",
				},
				DHCP: DHCPConfig{
					DHCPProvider: "none",
				},
				Inspector: InspectorConfig{
					RequireManagedBoot: boolPtr(false),
					PowerOff:           "true",
					ExtraKernelParams:  "ipa-inspection-collectors=default ipa-enable-vlan-interfaces=all ipa-inspection-dhcp-all-interfaces=1 ipa-collect-lldp=1",
					Hooks:              "$default_hooks,parse-lldp",
					AddPorts:           "all",
					KeepPorts:          "present",
				},
				AutoDiscovery: AutoDiscoveryConfig{
					Enabled: "false",
					Driver:  "ipmi",
				},
				IPMI: IPMIConfig{
					UseIpmitoolRetries:  boolPtr(false),
					MinCommandInterval:  5,
					CommandRetryTimeout: 60,
					CipherSuiteVersions: "3,17",
				},
				JSONRPC: JSONRPCConfig{
					AuthStrategy: "noauth",
					HostIP:       "::",
				},
				Nova: NovaConfig{
					SendPowerNotifications: boolPtr(false),
				},
				OsloMessagingNotifications: OsloMessagingNotificationsConfig{
					Driver:       "prometheus_exporter",
					Location:     "/shared/ironic_prometheus_exporter",
					TransportURL: "fake://",
				},
				SensorData: SensorDataConfig{
					SendSensorData: boolPtr(false),
					Interval:       160,
				},
				Metrics: MetricsConfig{
					Backend: "collector",
				},
				PXE: PXEConfig{
					BootRetryTimeout:      1200,
					ImagesPath:            "/shared/html/tmp",
					InstanceMasterPath:    "/shared/html/master_images",
					TFTPMasterPath:        "/shared/tftpboot/master_images",
					TFTPRoot:              "/shared/tftpboot",
					KernelAppendParams:    "nofb nomodeset vga=normal ipa-insecure=1 fips=1 sshkey=\"\" systemd.journald.forward_to_console=yes",
					EnableNetbootFallback: boolPtr(true),
					IPXEFallbackScript:    "inspector.ipxe",
					IPXEConfigTemplate:    "/templates/ipxe_config.template",
				},
				Redfish: RedfishConfig{
					UseSwift:           boolPtr(false),
					KernelAppendParams: "nofb nomodeset vga=normal ipa-insecure=1 fips=1 sshkey=\"\" systemd.journald.forward_to_console=yes",
				},
				ILO: ILOConfig{
					KernelAppendParams:    "nofb nomodeset vga=normal ipa-insecure=1 fips=1 sshkey=\"\" systemd.journald.forward_to_console=yes",
					UseWebServerForImages: boolPtr(true),
				},
				IRMC: IRMCConfig{
					KernelAppendParams: "nofb nomodeset vga=normal ipa-insecure=1 fips=1 sshkey=\"\" systemd.journald.forward_to_console=yes",
				},
				ServiceCatalog: ServiceCatalogConfig{
					EndpointOverride: "http://ironic:6385/v1/",
				},
				SSL: SSLConfig{
					CertFile: "",
					KeyFile:  "",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			err := c.Unmarshal(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !configEqual(*c, tt.want) {
				t.Errorf("Config.Unmarshal() got = %+v, want %+v", *c, tt.want)
			}
		})
	}
}

func TestConfig_Marshal(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "empty config",
			config:  Config{},
			wantErr: false,
		},
		{
			name: "config with DEFAULT section",
			config: Config{
				Default: DefaultConfig{
					AuthStrategy:         "keystone",
					Debug:                boolPtr(true),
					EnabledHardwareTypes: "ipmi,redfish",
				},
			},
			wantErr: false,
		},
		{
			name: "config with multiple sections",
			config: Config{
				API: APIConfig{
					HostIP: "127.0.0.1",
					Port:   6385,
				},
				Database: DatabaseConfig{
					Connection: "sqlite:///test.db",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.config.Marshal()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(data) == 0 && !isEmptyConfig(tt.config) {
				t.Errorf("Config.Marshal() returned empty data for non-empty config")
			}
		})
	}
}

func TestConfig_MarshalUnmarshal(t *testing.T) {
	original := Config{
		Default: DefaultConfig{
			AuthStrategy: "keystone",
			Debug:        boolPtr(true),
		},
		API: APIConfig{
			HostIP: "0.0.0.0",
			Port:   6385,
		},
		Database: DatabaseConfig{
			Connection: "mysql://user:pass@host/db",
		},
		IPMI: IPMIConfig{
			UseIpmitoolRetries:  boolPtr(false),
			MinCommandInterval:  5,
			CommandRetryTimeout: 60,
		},
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	var unmarshaled Config
	err = unmarshaled.Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	if !configEqual(original, unmarshaled) {
		t.Errorf("Marshal/Unmarshal roundtrip failed")
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func configEqual(a, b Config) bool {
	// Simple comparison for test purposes
	return configDefaultEqual(a.Default, b.Default) &&
		configAgentEqual(a.Agent, b.Agent) &&
		configAPIEqual(a.API, b.API) &&
		configConductorEqual(a.Conductor, b.Conductor) &&
		configDatabaseEqual(a.Database, b.Database) &&
		configDeployEqual(a.Deploy, b.Deploy) &&
		configDHCPEqual(a.DHCP, b.DHCP) &&
		configInspectorEqual(a.Inspector, b.Inspector) &&
		configAutoDiscoveryEqual(a.AutoDiscovery, b.AutoDiscovery) &&
		configIPMIEqual(a.IPMI, b.IPMI) &&
		configJSONRPCEqual(a.JSONRPC, b.JSONRPC) &&
		configNovaEqual(a.Nova, b.Nova) &&
		configOsloMessagingNotificationsEqual(
			a.OsloMessagingNotifications, b.OsloMessagingNotifications) &&
		configSensorDataEqual(a.SensorData, b.SensorData) &&
		configMetricsEqual(a.Metrics, b.Metrics) &&
		configPXEEqual(a.PXE, b.PXE) &&
		configRedfishEqual(a.Redfish, b.Redfish) &&
		configILOEqual(a.ILO, b.ILO) &&
		configIRMCEqual(a.IRMC, b.IRMC) &&
		configServiceCatalogEqual(a.ServiceCatalog, b.ServiceCatalog) &&
		configSSLEqual(a.SSL, b.SSL)
}

func configDefaultEqual(a, b DefaultConfig) bool {
	return a.AuthStrategy == b.AuthStrategy &&
		boolPtrEqual(a.Debug, b.Debug) &&
		a.DefaultDeployInterface == b.DefaultDeployInterface &&
		a.DefaultInspectInterface == b.DefaultInspectInterface &&
		a.DefaultNetworkInterface == b.DefaultNetworkInterface &&
		a.EnabledBiosInterfaces == b.EnabledBiosInterfaces &&
		a.EnabledBootInterfaces == b.EnabledBootInterfaces &&
		a.EnabledDeployInterfaces == b.EnabledDeployInterfaces &&
		a.EnabledFirmwareInterfaces == b.EnabledFirmwareInterfaces &&
		a.EnabledHardwareTypes == b.EnabledHardwareTypes &&
		a.EnabledInspectInterfaces == b.EnabledInspectInterfaces &&
		a.EnabledManagementInterfaces == b.EnabledManagementInterfaces &&
		a.EnabledNetworkInterfaces == b.EnabledNetworkInterfaces &&
		a.EnabledPowerInterfaces == b.EnabledPowerInterfaces &&
		a.EnabledRaidInterfaces == b.EnabledRaidInterfaces &&
		a.EnabledVendorInterfaces == b.EnabledVendorInterfaces &&
		a.RPCTransport == b.RPCTransport &&
		boolPtrEqual(a.UseStderr, b.UseStderr) &&
		a.HashRingAlgorithm == b.HashRingAlgorithm &&
		a.MyIP == b.MyIP &&
		a.Host == b.Host &&
		boolPtrEqual(a.WebserverVerifyCA, b.WebserverVerifyCA) &&
		a.IsolinuxBin == b.IsolinuxBin &&
		a.GrubConfigPath == b.GrubConfigPath
}

func configAgentEqual(a, b AgentConfig) bool {
	return a.DeployLogsCollect == b.DeployLogsCollect &&
		a.DeployLogsLocalPath == b.DeployLogsLocalPath &&
		a.MaxCommandAttempts == b.MaxCommandAttempts &&
		a.CertificatesPath == b.CertificatesPath
}

func configAPIEqual(a, b APIConfig) bool {
	return a.UnixSocket == b.UnixSocket &&
		a.UnixSocketMode == b.UnixSocketMode &&
		a.PublicEndpoint == b.PublicEndpoint &&
		a.HostIP == b.HostIP &&
		a.Port == b.Port &&
		boolPtrEqual(a.EnableSSLAPI, b.EnableSSLAPI) &&
		a.APIWorkers == b.APIWorkers
}

func configConductorEqual(a, b ConductorConfig) bool {
	return boolPtrEqual(a.AutomatedClean, b.AutomatedClean) &&
		a.DeployCallbackTimeout == b.DeployCallbackTimeout &&
		a.BootloaderByArch == b.BootloaderByArch &&
		a.VerifyStepPriorityOverride == b.VerifyStepPriorityOverride &&
		boolPtrEqual(a.NodeHistory, b.NodeHistory) &&
		a.PowerStateChangeTimeout == b.PowerStateChangeTimeout &&
		a.DeployKernel == b.DeployKernel &&
		a.DeployKernelByArch == b.DeployKernelByArch &&
		a.DeployRamdisk == b.DeployRamdisk &&
		a.DeployRamdiskByArch == b.DeployRamdiskByArch &&
		boolPtrEqual(a.DisableDeepImageInspection, b.DisableDeepImageInspection) &&
		a.FileURLAllowedPaths == b.FileURLAllowedPaths
}

func configDatabaseEqual(a, b DatabaseConfig) bool {
	return a.Connection == b.Connection &&
		boolPtrEqual(a.SqliteSynchronous, b.SqliteSynchronous)
}

func configDeployEqual(a, b DeployConfig) bool {
	return a.DefaultBootOption == b.DefaultBootOption &&
		a.EraseDevicesMetadataPriority == b.EraseDevicesMetadataPriority &&
		a.EraseDevicesPriority == b.EraseDevicesPriority &&
		a.HTTPRoot == b.HTTPRoot &&
		a.HTTPURL == b.HTTPURL &&
		boolPtrEqual(a.FastTrack, b.FastTrack) &&
		a.RamdiskImageDownloadSource == b.RamdiskImageDownloadSource &&
		a.ExternalHTTPURL == b.ExternalHTTPURL &&
		a.ExternalCallbackURL == b.ExternalCallbackURL
}

func configDHCPEqual(a, b DHCPConfig) bool {
	return a.DHCPProvider == b.DHCPProvider
}

func configInspectorEqual(a, b InspectorConfig) bool {
	return boolPtrEqual(a.RequireManagedBoot, b.RequireManagedBoot) &&
		a.PowerOff == b.PowerOff &&
		a.ExtraKernelParams == b.ExtraKernelParams &&
		a.Hooks == b.Hooks &&
		a.AddPorts == b.AddPorts &&
		a.KeepPorts == b.KeepPorts
}

func configAutoDiscoveryEqual(a, b AutoDiscoveryConfig) bool {
	return a.Enabled == b.Enabled &&
		a.Driver == b.Driver
}

func configIPMIEqual(a, b IPMIConfig) bool {
	return boolPtrEqual(a.UseIpmitoolRetries, b.UseIpmitoolRetries) &&
		a.MinCommandInterval == b.MinCommandInterval &&
		a.CommandRetryTimeout == b.CommandRetryTimeout &&
		a.CipherSuiteVersions == b.CipherSuiteVersions
}

func configJSONRPCEqual(a, b JSONRPCConfig) bool {
	return a.AuthStrategy == b.AuthStrategy &&
		a.HostIP == b.HostIP &&
		a.UnixSocket == b.UnixSocket &&
		a.UnixSocketMode == b.UnixSocketMode
}

func configNovaEqual(a, b NovaConfig) bool {
	return boolPtrEqual(a.SendPowerNotifications, b.SendPowerNotifications)
}

func configOsloMessagingNotificationsEqual(a, b OsloMessagingNotificationsConfig) bool {
	return a.Driver == b.Driver &&
		a.Location == b.Location &&
		a.TransportURL == b.TransportURL
}

func configSensorDataEqual(a, b SensorDataConfig) bool {
	return boolPtrEqual(a.SendSensorData, b.SendSensorData) &&
		a.Interval == b.Interval
}

func configMetricsEqual(a, b MetricsConfig) bool {
	return a.Backend == b.Backend
}

func configPXEEqual(a, b PXEConfig) bool {
	return a.BootRetryTimeout == b.BootRetryTimeout &&
		a.ImagesPath == b.ImagesPath &&
		a.InstanceMasterPath == b.InstanceMasterPath &&
		a.TFTPMasterPath == b.TFTPMasterPath &&
		a.TFTPRoot == b.TFTPRoot &&
		a.KernelAppendParams == b.KernelAppendParams &&
		boolPtrEqual(a.EnableNetbootFallback, b.EnableNetbootFallback) &&
		a.IPXEFallbackScript == b.IPXEFallbackScript &&
		a.IPXEConfigTemplate == b.IPXEConfigTemplate
}

func configRedfishEqual(a, b RedfishConfig) bool {
	return boolPtrEqual(a.UseSwift, b.UseSwift) &&
		a.KernelAppendParams == b.KernelAppendParams
}

func configILOEqual(a, b ILOConfig) bool {
	return a.KernelAppendParams == b.KernelAppendParams &&
		boolPtrEqual(a.UseWebServerForImages, b.UseWebServerForImages)
}

func configIRMCEqual(a, b IRMCConfig) bool {
	return a.KernelAppendParams == b.KernelAppendParams
}

func configServiceCatalogEqual(a, b ServiceCatalogConfig) bool {
	return a.EndpointOverride == b.EndpointOverride
}

func configSSLEqual(a, b SSLConfig) bool {
	return a.CertFile == b.CertFile &&
		a.KeyFile == b.KeyFile
}

func boolPtrEqual(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func isEmptyConfig(c Config) bool {
	return c.Default.AuthStrategy == "" &&
		c.API.HostIP == "" &&
		c.API.Port == 0 &&
		c.Database.Connection == "" &&
		c.Agent.DeployLogsCollect == "" &&
		c.Conductor.DeployCallbackTimeout == 0 &&
		c.Deploy.DefaultBootOption == "" &&
		c.DHCP.DHCPProvider == "" &&
		c.Inspector.PowerOff == "" &&
		c.AutoDiscovery.Enabled == "" &&
		c.IPMI.MinCommandInterval == 0 &&
		c.JSONRPC.AuthStrategy == "" &&
		c.JSONRPC.UnixSocket == "" &&
		c.OsloMessagingNotifications.Driver == "" &&
		c.SensorData.Interval == 0 &&
		c.Metrics.Backend == "" &&
		c.PXE.BootRetryTimeout == 0 &&
		c.Redfish.KernelAppendParams == "" &&
		c.ILO.KernelAppendParams == "" &&
		c.IRMC.KernelAppendParams == "" &&
		c.ServiceCatalog.EndpointOverride == "" &&
		c.SSL.CertFile == ""
}
