package config

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
	"github.com/spf13/viper"
)

const (
	ipxePatchDefault = "" // "set user-class iPXE"
	magicString      = "464vn90e7rbj08xbwdjejmdf4it17c5zfzjyfhthbh19eij201hjgit021bmpdb9ctrc87x2ymc8e7icu4ffi15x1hah9iyaiz38ckyap8hwx2vt5rm44ixv4hau8iw718q5yd019um5dt2xpqqa2rjtdypzr5v1gun8un110hhwp8cex7pqrh2ivh0ynpm4zkkwc8wcn367zyethzy7q8hzudyeyzx3cgmxqbkh825gcak7kxzjbgjajwizryv7ec1xm2h0hh7pz29qmvtgfjj1vphpgq1zcbiiehv52wrjy9yq473d9t1rvryy6929nk435hfx55du3ih05kn5tju3vijreru1p6knc988d4gfdz28eragvryq5x8aibe5trxd0t6t7jwxkde34v6pj1khmp50k6qqj3nzgcfzabtgqkmeqhdedbvwf3byfdma4nkv3rcxugaj2d0ru30pa2fqadjqrtjnv8bu52xzxv7irbhyvygygxu1nt5z4fh9w1vwbdcmagep26d298zknykf2e88kumt59ab7nq79d8amnhhvbexgh48e8qc61vq2e9qkihzt1twk1ijfgw70nwizai15iqyted2dt9gfmf2gg7amzufre79hwqkddc1cd935ywacnkrnak6r7xzcz7zbmq3kt04u2hg1iuupid8rt4nyrju51e6uejb2ruu36g9aibmz3hnmvazptu8x5tyxk820g2cdpxjdij766bt2n3djur7v623a2v44juyfgz80ekgfb9hkibpxh3zgknw8a34t4jifhf116x15cei9hwch0fye3xyq0acuym8uhitu5evc4rag3ui0fny3qg4kju7zkfyy8hwh537urd5uixkzwu5bdvafz4jmv7imypj543xg5em8jk8cgk7c4504xdd5e4e71ihaumt6u5u2t1w7um92fepzae8p0vq93wdrd1756npu1pziiur1payc7kmdwyxg3hj5n4phxbc29x0tcddamjrwt260b0w"
)

type UnifiConfig struct {
	APIKey   string `mapstructure:"api_key"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Endpoint string `mapstructure:"endpoint"`
	Site     string `mapstructure:"site"`
	Device   string `mapstructure:"device"`
	Insecure bool   `mapstructure:"insecure"`
}

type TftpConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	Address       string `mapstructure:"address"`
	Port          int    `mapstructure:"port"`
	RootDirectory string `mapstructure:"root_directory"`
	IpxePatch     string `mapstructure:"ipxe_patch"`
}

type IpxeUrl struct {
	Address string `mapstructure:"address"`
	Port    int    `mapstructure:"port"`
	Scheme  string `mapstructure:"scheme"`
	Path    string `mapstructure:"path"`
}

func (u IpxeUrl) GetUrl(paths ...string) *url.URL {
	path := u.Path
	if len(paths) > 0 {
		path = filepath.Join(paths...)
	}

	return &url.URL{
		Scheme: u.Scheme,
		Host: func() string {
			switch u.Scheme {
			case "http":
				if u.Port == 80 {
					return u.Address
				}
			case "https":
				if u.Port == 443 {
					return u.Address
				}
			}
			return fmt.Sprintf("%s:%d", u.Address, u.Port)
		}(),
		Path: path,
	}
}

type DhcpConfig struct {
	Enabled           bool     `mapstructure:"enabled"`
	Interface         string   `mapstructure:"interface"`
	Address           string   `mapstructure:"address"`
	Port              int      `mapstructure:"port"`
	ProxyEnabled      bool     `mapstructure:"proxy_enabled"`
	IpxeBinaryUrl     IpxeUrl  `mapstructure:"ipxe_binary_url"`
	IpxeHttpUrl       IpxeUrl  `mapstructure:"ipxe_http_url"`
	IpxeHttpScript    IpxeUrl  `mapstructure:"ipxe_http_script"`
	IpxeHttpScriptURL string   `mapstructure:"ipxe_http_script_url"`
	TftpAddress       string   `mapstructure:"tftp_address"`
	TftpPort          int      `mapstructure:"tftp_port"`
	SyslogIP          string   `mapstructure:"syslog_ip"`
	StaticIPAMEnabled bool     `mapstructure:"static_ipam_enabled"`
	LeaseFile         string   `mapstructure:"lease_file"`
	ConfigFile        string   `mapstructure:"config_file"`
	FallbackEnabled   bool     `mapstructure:"fallback_enabled"`
	FallbackIPStart   string   `mapstructure:"fallback_ip_start"`
	FallbackIPEnd     string   `mapstructure:"fallback_ip_end"`
	FallbackGateway   string   `mapstructure:"fallback_gateway"`
	FallbackSubnet    string   `mapstructure:"fallback_subnet"`
	FallbackDNS       []string `mapstructure:"fallback_dns"`
	FallbackDomain    string   `mapstructure:"fallback_domain"`
	FallbackNetboot   bool     `mapstructure:"fallback_netboot"`
}

type IpxeHttpScript struct {
	Enabled               bool     `mapstructure:"enabled"`
	Retries               int      `mapstructure:"retries"`
	RetryDelay            int      `mapstructure:"retry_delay"`
	TinkServer            string   `mapstructure:"tink_server"`
	HookURL               string   `mapstructure:"hook_url"`
	TinkServerInsecureTLS bool     `mapstructure:"tink_server_insecure_tls"`
	TinkServerUseTLS      bool     `mapstructure:"tink_server_use_tls"`
	ExtraKernelArgs       []string `mapstructure:"extra_kernel_args"`
	StaticIPXEEnabled     bool     `mapstructure:"static_ipxe_enabled"`
	StaticFilesEnabled    bool     `mapstructure:"static_files_enabled"`
}

type IsoConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Url         string `mapstructure:"url"`
	MagicString string `mapstructure:"magic_string"`
}

type OtelConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	Insecure bool   `mapstructure:"insecure"`
}

type ImageURL struct {
	Path string `mapstructure:"path"`
	URL  string `mapstructure:"url"`
}

type StaticConfig struct {
	Enabled       bool       `mapstructure:"enabled"`
	ImageURLs     []ImageURL `mapstructure:"image_urls"`
	RootDirectory string     `mapstructure:"root_directory"`
}

type DnsmasqConfig struct {
	Enabled           bool     `mapstructure:"enabled"`
	RootDirectory     string   `mapstructure:"root_directory"`
	TFTPServer        string   `mapstructure:"tftp_server"`
	HTTPServer        string   `mapstructure:"http_server"`
	AutoAssignEnabled bool     `mapstructure:"auto_assign_enabled"`
	IPPoolStart       string   `mapstructure:"ip_pool_start"`
	IPPoolEnd         string   `mapstructure:"ip_pool_end"`
	DefaultLeaseTime  uint32   `mapstructure:"default_lease_time"`
	DefaultGateway    string   `mapstructure:"default_gateway"`
	DefaultSubnet     string   `mapstructure:"default_subnet"`
	DefaultDNS        []string `mapstructure:"default_dns"`
	DefaultDomain     string   `mapstructure:"default_domain"`
}

type Config struct {
	Address         string         `mapstructure:"address"`
	Port            int            `mapstructure:"port"`
	Unifi           UnifiConfig    `mapstructure:"unifi"`
	Tftp            TftpConfig     `mapstructure:"tftp"`
	Dhcp            DhcpConfig     `mapstructure:"dhcp"`
	LogLevel        string         `mapstructure:"log_level"`
	BackendFilePath string         `mapstructure:"backend_file_path"`
	Log             logr.Logger    `mapstructure:"-"`
	Iso             IsoConfig      `mapstructure:"iso"`
	IpxeHttpScript  IpxeHttpScript `mapstructure:"ipxe_http_script"`
	TrustedProxies  string         `mapstructure:"trusted_proxies"`
	Otel            OtelConfig     `mapstructure:"otel"`
	Static          StaticConfig   `mapstructure:"static"`
	Dnsmasq         DnsmasqConfig  `mapstructure:"dnsmasq"`
	ResetDelaySec   int            `mapstructure:"reset_delay_sec"`
	FirmwarePath    string         `mapstructure:"firmware_path"`
}

func (c *Config) GetIpxeHttpUrl() (*url.URL, error) {
	if c.Dhcp.IpxeHttpScriptURL != "" {
		return url.Parse(c.Dhcp.IpxeHttpScriptURL)
	} else {
		return c.Dhcp.IpxeHttpUrl.GetUrl(), nil
	}
}

func (c *Config) GetOsieUrl() (*url.URL, error) {
	if c.IpxeHttpScript.HookURL != "" {
		return url.Parse(c.IpxeHttpScript.HookURL)
	} else {
		return c.Dhcp.IpxeHttpUrl.GetUrl("images"), nil
	}
}

type defaultNetworkInfo struct {
	BindIP     string
	ExternalIP string
	Iface      string
	Port       int
}

func GetDefaultIpAddrInfo() defaultNetworkInfo {
	res := defaultNetworkInfo{}
	addr, iface, err := GetLocalIP()
	if err != nil {
		if addr, ok := os.LookupEnv("EXTERNAL_IP"); !ok {
			res.ExternalIP = "127.0.0.1"
		} else {
			res.ExternalIP = addr
		}
		if iface, ok := os.LookupEnv("INTERFACE"); !ok {
			res.Iface = "eth0"
		} else {
			res.Iface = iface
		}
	} else {
		res.ExternalIP = addr
		res.Iface = iface
	}

	if port, ok := os.LookupEnv("PORT"); !ok {
		res.Port = 8080
	} else {
		res.Port, _ = strconv.Atoi(port)
	}

	if bindIP, ok := os.LookupEnv("BIND_IP"); !ok {
		res.BindIP = "0.0.0.0"
	} else {
		res.BindIP = bindIP
	}

	return res
}

func NewConfig() (conf *Config, err error) {
	conf = &Config{}

	netInfo := GetDefaultIpAddrInfo()

	viper.SetConfigName("config")

	viper.AddConfigPath("/app/")
	viper.AddConfigPath("/config/")

	wd, _ := os.Getwd()
	confDir := "."
	if wd == "/" {
		confDir = os.TempDir()
		if configDir, err := os.UserConfigDir(); err != nil {
			log.Fatalf("Unable to get user config dir: %s", err.Error())
		} else {
			confDir = configDir
		}
	}

	if _, err := os.Stat(confDir); errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(confDir, 0o755); err != nil {
			log.Fatalf("Unable to create config directory: %s", err.Error())
		}
	}

	viper.AddConfigPath(confDir)

	viper.SetDefault("reset_delay_sec", 45)

	viper.SetDefault("address", netInfo.BindIP)
	viper.SetDefault("port", netInfo.Port)
	viper.SetDefault("trusted_proxies", "")
	viper.SetDefault("backend_file_path", "backend.yaml")

	viper.SetDefault("unifi.endpoint", "https://10.0.0.1")
	viper.SetDefault("unifi.site", "default")
	viper.SetDefault("unifi.device", "")
	viper.SetDefault("unifi.insecure", true)
	viper.SetDefault("unifi.api_key", "your_api_key")

	viper.SetDefault("tftp.enabled", false)
	viper.SetDefault("tftp.address", netInfo.BindIP)
	viper.SetDefault("tftp.port", 69)
	viper.SetDefault("tftp.root_directory", "/tftpboot")
	viper.SetDefault("tftp.ipxe_patch", ipxePatchDefault)

	viper.SetDefault("dhcp.enabled", false)
	viper.SetDefault("dhcp.interface", netInfo.Iface)
	viper.SetDefault("dhcp.address", netInfo.BindIP)
	viper.SetDefault("dhcp.port", 67)
	viper.SetDefault("dhcp.proxy_enabled", false)
	viper.SetDefault("dhcp.ipxe_http_script_url", "")
	viper.SetDefault("dhcp.ipxe_binary_url.address", netInfo.ExternalIP)
	viper.SetDefault("dhcp.ipxe_binary_url.port", netInfo.Port)
	viper.SetDefault("dhcp.ipxe_binary_url.scheme", "http")
	viper.SetDefault("dhcp.ipxe_binary_url.path", "/ipxe/")
	viper.SetDefault("dhcp.ipxe_http_url.address", netInfo.ExternalIP)
	viper.SetDefault("dhcp.ipxe_http_url.port", netInfo.Port)
	viper.SetDefault("dhcp.ipxe_http_url.scheme", "http")
	viper.SetDefault("dhcp.ipxe_http_url.path", "/auto.ipxe")
	viper.SetDefault("dhcp.tftp_address", netInfo.ExternalIP)
	viper.SetDefault("dhcp.tftp_port", 69)
	viper.SetDefault("dhcp.syslog_ip", "")
	viper.SetDefault("dhcp.lease_file", "")
	viper.SetDefault("dhcp.static_ipam_enabled", false)
	viper.SetDefault("dhcp.fallback_enabled", false)
	viper.SetDefault("dhcp.fallback_ip_start", "192.168.1.100")
	viper.SetDefault("dhcp.fallback_ip_end", "192.168.1.200")
	viper.SetDefault("dhcp.fallback_gateway", "192.168.1.1")
	viper.SetDefault("dhcp.fallback_subnet", "255.255.255.0")
	viper.SetDefault("dhcp.fallback_dns", []string{"8.8.8.8", "8.8.4.4"})
	viper.SetDefault("dhcp.fallback_domain", "local")
	viper.SetDefault("dhcp.fallback_netboot", false)

	viper.SetDefault("static.enabled", true)
	viper.SetDefault("static.image_urls", []ImageURL{})
	viper.SetDefault("static.root_directory", "/shared/html")

	viper.SetDefault("dnsmasq.enabled", true)
	viper.SetDefault("dnsmasq.root_directory", "/shared/dnsmasq")
	viper.SetDefault("dnsmasq.tftp_server", netInfo.ExternalIP)
	viper.SetDefault("dnsmasq.http_server", netInfo.ExternalIP)
	viper.SetDefault("dnsmasq.auto_assign_enabled", false)
	viper.SetDefault("dnsmasq.ip_pool_start", "192.168.1.100")
	viper.SetDefault("dnsmasq.ip_pool_end", "192.168.1.200")
	viper.SetDefault("dnsmasq.default_lease_time", 604800) // 1 week
	viper.SetDefault("dnsmasq.default_gateway", "192.168.1.1")
	viper.SetDefault("dnsmasq.default_subnet", "255.255.255.0")
	viper.SetDefault("dnsmasq.default_dns", []string{"8.8.8.8", "8.8.4.4"})
	viper.SetDefault("dnsmasq.default_domain", "local")

	viper.SetDefault("ipxe_http_script.enabled", true)
	viper.SetDefault("ipxe_http_script.retries", 3)
	viper.SetDefault("ipxe_http_script.retry_delay", 5)
	viper.SetDefault("ipxe_http_script.tink_server", "")
	viper.SetDefault("ipxe_http_script.hook_url", "")
	viper.SetDefault("ipxe_http_script.tink_server_insecure_tls", true)
	viper.SetDefault("ipxe_http_script.tink_server_use_tls", false)
	viper.SetDefault("ipxe_http_script.extra_kernel_args", []string{})
	viper.SetDefault("ipxe_http_script.static_ipxe_enabled", false)
	viper.SetDefault("ipxe_http_script.static_files_enabled", false)

	viper.SetDefault("otel.endpoint", "")
	viper.SetDefault("otel.insecure", true)

	viper.SetDefault("iso.enabled", true)
	viper.SetDefault("iso.url", "")
	viper.SetDefault("iso.magic_string", magicString)

	viper.SetDefault("log_level", "info")

	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			configFile := filepath.Join(confDir, "config.yaml")
			log.Printf("config: creating default config file: %s", configFile)
			if err := viper.SafeWriteConfigAs(configFile); err != nil {
				log.Fatalf("Unable to write config file: %s", err.Error())
			}
			if err := viper.ReadInConfig(); err != nil {
				log.Fatalf("Unable to read after writing config file: %s", err.Error())
			}
		} else {
			log.Fatalf("Unable to read config file: %s", err.Error())
		}
	}

	for _, key := range viper.AllKeys() {
		envKey := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
		err := viper.BindEnv(key, envKey)
		if err != nil {
			log.Fatalf("config: unable to bind env: %s", err.Error())
		}
	}

	conf.Log = defaultLogger(conf.LogLevel)

	// Load the Config the first time we start the app.
	err = loadConfig(conf)
	if err != nil {
		return conf, err
	}

	// Tell viper to watch the config file.
	viper.WatchConfig()

	// Tell viper what to do when it detects the
	// config file has changed.
	viper.OnConfigChange(func(_ fsnotify.Event) {
		_ = loadConfig(conf)
	})

	return conf, err
}

func loadConfig(conf *Config) (err error) {
	// read the config file into viper and
	// handle (ignore the file) any errors
	err = viper.MergeInConfig()
	if err != nil {
		return nil
	}

	err = viper.Unmarshal(conf)
	if err != nil {
		return
	}

	return
}

func GetLocalIP() (string, string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", "", err
	}
	// handle err
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return "", "", err
		}

		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if !ip.IsLoopback() {
				if ip.To4() != nil {
					return ip.String(), i.Name, nil
				}
			}
		}
	}

	return "", "", nil
}

// defaultLogger uses the slog logr implementation.
func defaultLogger(level string) logr.Logger {
	// source file and function can be long. This makes the logs less readable.
	// truncate source file and function to last 3 parts for improved readability.
	customAttr := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.SourceKey {
			ss, ok := a.Value.Any().(*slog.Source)
			if !ok || ss == nil {
				return a
			}
			f := strings.Split(ss.Function, "/")
			if len(f) > 3 {
				ss.Function = filepath.Join(f[len(f)-3:]...)
			}
			p := strings.Split(ss.File, "/")
			if len(p) > 3 {
				ss.File = filepath.Join(p[len(p)-3:]...)
			}

			return a
		}

		return a
	}
	opts := &slog.HandlerOptions{AddSource: true, ReplaceAttr: customAttr}
	switch level {
	case "debug":
		opts.Level = slog.LevelDebug
	default:
		opts.Level = slog.LevelInfo
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, opts))

	return logr.FromSlogHandler(log.Handler())
}
