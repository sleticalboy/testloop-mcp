package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseServerConfigDefaults(t *testing.T) {
	var stderr bytes.Buffer

	cfg, code := parseServerConfig(nil, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if cfg.transport != "stdio" || cfg.addr != ":8080" || cfg.stateless {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestParseServerConfigHTTP(t *testing.T) {
	var stderr bytes.Buffer

	cfg, code := parseServerConfig([]string{"--transport=http", "--addr=:18080", "--stateless"}, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if cfg.transport != "http" || cfg.addr != ":18080" || !cfg.stateless {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestParseServerConfigInvalidFlag(t *testing.T) {
	var stderr bytes.Buffer

	_, code := parseServerConfig([]string{"--bad"}, &stderr)

	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("stderr missing flag error: %q", stderr.String())
	}
}

func TestParseServerConfigRejectsUnsupportedTransport(t *testing.T) {
	var stderr bytes.Buffer

	cfg, code := parseServerConfig([]string{"--transport=grpc"}, &stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if cfg.transport != "grpc" {
		t.Fatalf("transport = %q, want grpc", cfg.transport)
	}
	if !strings.Contains(stderr.String(), "不支持的传输模式") {
		t.Fatalf("stderr missing transport error: %q", stderr.String())
	}
}

func TestHTTPMuxHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	newHTTPMux(newTestloopServer(), false).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok\n" {
		t.Fatalf("body = %q, want ok", rec.Body.String())
	}
}
