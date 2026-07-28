package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Server is a Docker Hub reverse-proxy mirror.
type Server struct {
	listenAddr string
	proxy      http.Handler
	srv        *http.Server
	logger     Logger
}

// Logger provides logging capabilities.
type Logger interface {
	Printf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Println(v ...interface{})
}

// FileLogger writes logs to a file.
type FileLogger struct {
	logger *log.Logger
	mu     sync.RWMutex
}

// NewLogger creates a new Logger based on configuration.
// If LogDir is set, writes to a file; otherwise logs to stdout.
func NewLogger(cfg *Config) Logger {
	if cfg.LogDir != "" {
		logPath := fmt.Sprintf("%s/proxy.log", cfg.LogDir)
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			// Fallback to stdout if file cannot be opened
			return NewStdLogger()
		}
		return &FileLogger{
			logger: log.New(f, "[proxy] ", log.Ldate|log.Ltime|log.Lshortfile),
		}
	}
	return NewStdLogger()
}

// NewStdLogger creates a logger that writes to stdout.
func NewStdLogger() Logger {
	return &FileLogger{
		logger: log.New(os.Stdout, "[proxy] ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (l *FileLogger) Printf(format string, v ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.logger.Printf(format, v...)
}

func (l *FileLogger) Errorf(format string, v ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.logger.Printf("[ERROR] "+format, v...)
}

func (l *FileLogger) Println(v ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.logger.Println(v...)
}

// NewServer creates a new Docker Hub proxy server from config.
func NewServer(cfg *Config) (*Server, error) {
	up, err := url.Parse(cfg.Upstream)
	if err != nil {
		return nil, fmt.Errorf("parse upstream %q: %w", cfg.Upstream, err)
	}
	if !strings.HasSuffix(up.Path, "/") {
		up.Path += "/"
	}

	// Create logger based on configuration
	logger := NewLogger(cfg)

	// Create HTTP transport, with upstream proxy if configured
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment, // 默认使用 HTTP_PROXY/HTTPS_PROXY env var
	}
	if cfg.HTTPProxy != "" {
		proxyURL, err := url.Parse(cfg.HTTPProxy)
		if err != nil {
			return nil, fmt.Errorf("parse proxy URL %q: %w", cfg.HTTPProxy, err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	// Wrap transport with TLS config if needed (for HTTPS upstream)
	// 使用默认的 TLS 配置
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, address)
	}

	director := buildDirector(up, cfg.ListenAddr, logger)

	proxy := &httputil.ReverseProxy{
		Director:       director,
		ModifyResponse: modifyResponse(),
		ErrorHandler:   errorHandler,
		Transport:      transport, // 关键：设置带代理的 Transport
	}

	// Wrap proxy to handle health check
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK\n")
			return
		}
		proxy.ServeHTTP(w, r)
	})

	s := &Server{
		listenAddr: cfg.ListenAddr,
		proxy:      handler,
		srv: &http.Server{
			Addr:    cfg.ListenAddr,
			Handler: handler,
		},
		logger: logger,
	}

	return s, nil
}

// Start blocks until the server is shut down.
func (s *Server) Start() error {
	s.logger.Printf("Docker proxy listening on %s", s.listenAddr)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Errorf("listen: %w", err)
		return err
	}
	s.logger.Println("Server shutdown gracefully")
	return nil
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown() error {
	return s.srv.Close()
}

// buildDirector rewrites requests to point at the upstream registry.
func buildDirector(up *url.URL, listenAddr string, logger Logger) func(req *http.Request) {
	return func(req *http.Request) {
		req.Header.Set("X-Forwarded-For", req.RemoteAddr)

		for _, h := range []string{"User-Agent", "Accept", "Accept-Encoding"} {
			if v := req.Header.Get(h); v != "" {
				req.Header.Set(h, v)
			}
		}

		path := strings.TrimPrefix(req.URL.Path, "/")
		// Skip rewriting for special paths like /tokens
		if path != "" && !strings.HasPrefix(path, "v2/") && !strings.HasPrefix(path, "tokens") {
			path = "v2/" + path
		}

		req.URL = &url.URL{
			Scheme:   up.Scheme,
			Host:     up.Host,
			Path:     "/" + path,
			RawQuery: req.URL.RawQuery,
		}
		req.Host = up.Host

		logger.Printf("PROXY %s %s -> %s", req.Method, req.URL.Path, up.String())
	}
}

// modifyResponse rewrites manifest bodies so that blob URLs point back
// to the local proxy instead of the upstream Docker Hub.
func modifyResponse() func(resp *http.Response) error {
	return func(resp *http.Response) error {
		ct := resp.Header.Get("Content-Type")

		switch {
		case strings.HasPrefix(ct, "application/vnd.docker.distribution.manifest.v1"):
			rewriteManifestV1(resp.Body, resp)
		case strings.HasPrefix(ct, "application/vnd.docker.distribution.manifest.v2+json"):
			rewriteManifestV2(resp.Body, resp)
		default:
			return nil
		}
		return nil
	}
}

type v1Manifest struct {
	ID           string                  `json:"id,omitempty"`
	Architecture string                  `json:"architecture"`
	FsLayers     []v1FsLayer             `json:"fsLayers"`
	History      []v1History             `json:"history"`
	Name         string                  `json:"name"`
	Tag          string                  `json:"tag"`
	Repositories map[string][]string     `json:"repositories,omitempty"`
}

type v1FsLayer struct {
	BlobSum string `json:"blobSum"`
}

type v1History struct {
	V1Compatibility string `json:"v1Compatibility"`
}

type v2Manifest struct {
	SchemaVersion int            `json:"schemaVersion"`
	MediaType     string         `json:"mediaType,omitempty"`
	Config        v2Descriptor   `json:"config"`
	Layers        []v2Descriptor `json:"layers"`
}

type v2Descriptor struct {
	MediaType string   `json:"mediaType"`
	Size      int64    `json:"size"`
	Digest    string   `json:"digest"`
	Data      string   `json:"data,omitempty"`
	Urls      []string `json:"urls,omitempty"`
}

func rewriteManifestV1(body io.ReadCloser, resp *http.Response) {
	defer body.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, body); err != nil {
		return
	}

	var m v1Manifest
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		return
	}

	for i, h := range m.History {
		m.History[i].V1Compatibility = rewriteUpstreamURLs(h.V1Compatibility)
	}

	out, err := json.Marshal(m)
	if err != nil {
		return
	}
	resp.Body = io.NopCloser(bytes.NewReader(out))
	resp.ContentLength = int64(len(out))
}

func rewriteManifestV2(body io.ReadCloser, resp *http.Response) {
	defer body.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, body); err != nil {
		return
	}

	var m v2Manifest
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		return
	}

	for i, layer := range m.Layers {
		for j, u := range layer.Urls {
			layer.Urls[j] = rewriteUpstreamURLs(u)
		}
		m.Layers[i] = layer
	}

	out, err := json.Marshal(m)
	if err != nil {
		return
	}
	resp.Body = io.NopCloser(bytes.NewReader(out))
	resp.ContentLength = int64(len(out))
}

func rewriteUpstreamURLs(s string) string {
	return strings.NewReplacer(
		"https://index.docker.io/v1/", "",
		"https://registry-1.docker.io/", "",
		"https://auth.docker.io/token", "",
	).Replace(s)
}

func errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	fmt.Fprintf(w, "502 Bad Gateway: %v\n", err)
}