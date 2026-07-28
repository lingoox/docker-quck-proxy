package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"strings"

	"net/http"

	"docker-quck-proxy/internal/proxy"
)

// loadEnvFromFile 加载 .env 文件到环境变量
func loadEnvFromFile(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return // 文件不存在或不可读，静默失败
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		// 匹配 KEY=VALUE 格式
		if idx := strings.Index(line, "="); idx != -1 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			if key != "" {
				os.Setenv(key, value)
			}
		}
	}
}

func main() {
	// 加载 .env 文件（开发环境用）
	loadEnvFromFile(".env")
	fmt.Println("📁 已加载 .env 配置（如果存在）")

	cfg := proxy.DefaultConfig()

	if up := os.Getenv("UPSTREAM"); up != "" {
		cfg.Upstream = up
	}
	if ln := os.Getenv("LISTEN_ADDR"); ln != "" {
		cfg.ListenAddr = ln
	}
	if logDir := os.Getenv("LOG_DIR"); logDir != "" {
		cfg.LogDir = logDir
	}

	fmt.Printf("[DEBUG] LogDir=%q, Upstream=%q, Listen=%q\n", cfg.LogDir, cfg.Upstream, cfg.ListenAddr)

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
