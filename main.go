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

// 定义命令行标志（默认值为空/零，表示未设置）
var (
	upstream   = flag.String("upstream", "", "Upstream Docker Registry (empty to use default)")
	listen     = flag.String("port", "", "Listen address (empty to use default :5000)")
	logDir     = flag.String("logdir", "", "Log directory (empty to use stdout)")
	httpProxy  = flag.String("http-proxy", "", "Upstream HTTP proxy for accessing Docker Hub")
	httpsProxy = flag.String("https-proxy", "", "Upstream HTTPS proxy for accessing Docker Hub")
	logEnabled = flag.Bool("log-enabled", false, "Enable logging (default: false)")
)

func main() {
	flag.Parse() // 解析命令行参数

	cfg := proxy.DefaultConfig()

	// ==========================================
	// Upstream: flag (if set) > env var > default from DefaultConfig()
	// ==========================================
	if *upstream != "" {
		cfg.Upstream = *upstream
	} else if up := os.Getenv("UPSTREAM"); up != "" {
		cfg.Upstream = up
	}

	// ==========================================
	// ListenAddr: flag (if non-empty) > env var > default :5000
	// ==========================================
	if *listen != "" {
		cfg.ListenAddr = *listen
	} else if ln := os.Getenv("LISTEN_ADDR"); ln != "" {
		cfg.ListenAddr = ln
	}
	// else: keep default from DefaultConfig() (which is :5000)

	// ==========================================
	// LogDir: flag > env var > empty (stdout)
	// ==========================================
	if *logDir != "" {
		cfg.LogDir = *logDir
	} else if ld := os.Getenv("LOG_DIR"); ld != "" {
		cfg.LogDir = ld
	}

	// ==========================================
	// HTTP_PROXY / HTTPS_PROXY: flag > env var > empty
	// ==========================================
	if *httpProxy != "" {
		cfg.HTTPProxy = *httpProxy
	} else if hp := os.Getenv("HTTP_PROXY"); hp != "" {
		cfg.HTTPProxy = hp
	} else if hps := os.Getenv("HTTPS_PROXY"); hps != "" {
		cfg.HTTPProxy = hps
	}

	// ==========================================
	// LOG_ENABLED: check if flag was explicitly used via flag.Visit
	// ==========================================
	logEnabledSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "log-enabled" {
			logEnabledSet = true
		}
	})
	if logEnabledSet {
		cfg.LogEnabled = *logEnabled
	} else if le := os.Getenv("LOG_ENABLED"); le != "" {
		switch strings.ToLower(le) {
		case "true", "1", "y", "yes":
			cfg.LogEnabled = true
		case "false", "0", "n", "no":
			cfg.LogEnabled = false
		default:
			cfg.LogEnabled = false
		}
	}
	// else: keep default from DefaultConfig() (false)

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
