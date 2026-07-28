package proxy

// Config holds the server configuration.
type Config struct {
	ListenAddr  string `mapstructure:"listen_addr"`
	Upstream    string `mapstructure:"upstream"`
	LogDir      string `mapstructure:"log_dir"`
	HTTPProxy   string `mapstructure:"http_proxy"` // Upstream proxy for requests to Docker Hub
	LogEnabled  bool   `mapstructure:"log_enabled"`    // Default: false (logs disabled)
}

// DefaultConfig returns a Config with sensible defaults for a Docker Hub mirror.
func DefaultConfig() *Config {
	return &Config{
		ListenAddr: ":5000",
		Upstream:   "https://registry-1.docker.io",
		LogEnabled: false, // 默认关闭日志
	}
}
