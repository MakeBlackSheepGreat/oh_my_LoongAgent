package providers

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestHealthConfig 构造测试用 ProviderConfig。
func newTestHealthConfig(baseURL, apiKeyEnv string) ProviderConfig {
	return ProviderConfig{
		ProviderID: "test",
		BaseURL:    baseURL,
		ModelID:    "test-model",
		APIKeyEnv:  apiKeyEnv,
	}
}

// TestHealthCheck_OK 验证 /models 返回 200 时 OK=true、LatencyMs>0。
func TestHealthCheck_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("期望路径 /models，得到 %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv("HARNESS_TEST_API_KEY", "test-secret")
	config := newTestHealthConfig(srv.URL, "HARNESS_TEST_API_KEY")
	result := HealthCheck(context.Background(), config)
	if !result.OK {
		t.Fatalf("期望 OK，得到错误: %s", result.Error)
	}
	if result.LatencyMs <= 0 {
		t.Fatalf("期望正延迟，得到 %d", result.LatencyMs)
	}
}

// TestHealthCheck_Unreachable 验证指向不存在端口时 OK=false、Error 包含 "connection failed"。
func TestHealthCheck_Unreachable(t *testing.T) {
	// 找一个空闲端口后关闭，确保端口无人监听
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	t.Setenv("HARNESS_TEST_API_KEY", "test-secret")
	config := newTestHealthConfig("http://"+addr, "HARNESS_TEST_API_KEY")
	result := HealthCheck(context.Background(), config)
	if result.OK {
		t.Fatal("期望不可达时 OK=false")
	}
	if !strings.Contains(result.Error, "connection failed") {
		t.Fatalf("期望错误包含 'connection failed'，得到: %s", result.Error)
	}
}

// TestHealthCheck_Timeout 验证 mock 延迟导致 context 超时时 OK=false、Error="timeout"。
func TestHealthCheck_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(5 * time.Second):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
			return
		}
	}))
	defer srv.Close()

	t.Setenv("HARNESS_TEST_API_KEY", "test-secret")
	config := newTestHealthConfig(srv.URL, "HARNESS_TEST_API_KEY")
	result := HealthCheck(context.Background(), config)
	if result.OK {
		t.Fatal("期望超时时 OK=false")
	}
	if result.Error != "timeout" {
		t.Fatalf("期望 'timeout'，得到: %s", result.Error)
	}
}

// TestHealthCheck_HTTPError 验证 500 响应时 OK=false、Error 包含 "HTTP 500"。
func TestHealthCheck_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	t.Setenv("HARNESS_TEST_API_KEY", "test-secret")
	config := newTestHealthConfig(srv.URL, "HARNESS_TEST_API_KEY")
	result := HealthCheck(context.Background(), config)
	if result.OK {
		t.Fatal("期望 HTTP 错误时 OK=false")
	}
	if !strings.Contains(result.Error, "HTTP 500") {
		t.Fatalf("期望错误包含 'HTTP 500'，得到: %s", result.Error)
	}
}

// TestHealthCheck_NoAPIKey 验证不设环境变量时 OK=false。
func TestHealthCheck_NoAPIKey(t *testing.T) {
	config := newTestHealthConfig("http://127.0.0.1:8080", "HARNESS_TEST_API_KEY_NOT_SET")
	result := HealthCheck(context.Background(), config)
	if result.OK {
		t.Fatal("密钥缺失时期望 OK=false")
	}
}
