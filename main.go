package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"net/http"

	"docker-quck-proxy/internal/proxy"
)

// 定义命令行标志
var (
	upstream   = flag.String("upstream", "", "Upstream Docker Registry (override default)")
	listen     = flag.String("P", "", "Listen address (short form: -P), also use --listen")
	logDir     = flag.String("logdir", "", "Log directory (empty to use stdout)")
	httpProxy  = flag.String("http-proxy", "", "Upstream HTTP proxy for accessing Docker Hub")
	httpsProxy = flag.String("https-proxy", "", "Upstream HTTPS proxy for accessing Docker Hub")
	logEnabled = flag.Bool("log-enabled", false, "Enable logging (default: false)")
)

func main() {
	flag.Parse() // 解析命令行参数

	cfg := proxy.DefaultConfig()

	// 优先级：flag > env var > default
	// Upstream: flag 优先，否则 env var
	if *upstream != "" {
		cfg.Upstream = *upstream
	} else if up := os.Getenv("UPSTREAM"); up != "" {
		cfg.Upstream = up
	}

	// ListenAddr: flag 优先，否则 env var
	if *listen != "" {
		cfg.ListenAddr = *listen
	} else if ln := os.Getenv("LISTEN_ADDR"); ln != "" {
		cfg.ListenAddr = ln
	}

	// LogDir: flag 优先，否则 env var
	if *logDir != "" {
		cfg.LogDir = *logDir
	} else if ld := os.Getenv("LOG_DIR"); ld != "" {
		cfg.LogDir = ld
	}

	// HTTP_PROXY: flag 优先，否则 env var
	if *httpProxy != "" {
		cfg.HTTPProxy = *httpProxy
	} else if hp := os.Getenv("HTTP_PROXY"); hp != "" {
		cfg.HTTPProxy = hp
	} else if hps := os.Getenv("HTTPS_PROXY"); hps != "" && cfg.HTTPProxy == "" {
		// fallback to HTTPS_PROXY if HTTP_PROXY not set
		cfg.HTTPProxy = hps
	}

	// LOG_ENABLED: flag 优先，否则 env var
	if *logEnabled != false { // flag was explicitly set
		cfg.LogEnabled = *logEnabled
	} else if le := os.Getenv("LOG_ENABLED"); le != "" {
		switch strings.ToLower(le) {
		case "true", "1", "y", "yes":
			cfg.LogEnabled = true
		case "false", "0", "n", "no":
			cfg.LogEnabled = false
		default:
			cfg.LogEnabled = false // default
		}
	}

	fmt.Printf("[DEBUG] LogDir=%q, LogEnabled=%v, HTTPProxy=%q, Upstream=%q, Listen=%q\n",
		cfg.LogDir, cfg.LogEnabled, cfg.HTTPProxy, cfg.Upstream, cfg.ListenAddr)

	srv, err := proxy.NewServer(cfg)
	if err != nil {
		panic(err)
	}

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	if err := srv.Shutdown(); err != nil {
		panic(err)
	}

	fmt.Println("Shutdown complete")
}
