package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type demoInfoEnvelope struct {
	Code int              `json:"code"`
	Data demoInfoResponse `json:"data"`
}

func requestDemoInfo(t *testing.T, remoteAddr, host string) (*httptest.ResponseRecorder, demoInfoEnvelope) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/demo-info", demoInfo)
	req := httptest.NewRequest(http.MethodGet, "/demo-info", nil)
	req.RemoteAddr = remoteAddr
	req.Host = host
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	var body demoInfoEnvelope
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return resp, body
}

func TestDemoInfoDisabledByDefault(t *testing.T) {
	t.Setenv("SCENIC_GUIDE_DEMO_MODE", "")
	t.Setenv("SCENIC_GUIDE_DEMO_PASSWORD", "")

	resp, body := requestDemoInfo(t, "127.0.0.1:12345", "localhost:8080")
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	if body.Data.Enabled {
		t.Fatal("demo info unexpectedly enabled")
	}
	if len(body.Data.Accounts) != 0 {
		t.Fatalf("accounts = %d, want 0", len(body.Data.Accounts))
	}
}

func TestDemoInfoReturnsAccountsForLocalDemoMode(t *testing.T) {
	t.Setenv("SCENIC_GUIDE_DEMO_MODE", "true")
	t.Setenv("SCENIC_GUIDE_DEMO_PASSWORD", "ScenicDemo123456")

	resp, body := requestDemoInfo(t, "[::1]:12345", "[::1]:8080")
	if !body.Data.Enabled {
		t.Fatal("demo info is disabled")
	}
	if len(body.Data.Accounts) != 2 {
		t.Fatalf("accounts = %d, want 2", len(body.Data.Accounts))
	}
	if body.Data.Accounts[0].Username != "visitor" || body.Data.Accounts[1].Username != "admin" {
		t.Fatalf("unexpected accounts: %#v", body.Data.Accounts)
	}
	if body.Data.Accounts[0].Password != "ScenicDemo123456" || body.Data.Accounts[1].Password != "ScenicDemo123456" {
		t.Fatal("demo account password does not match configured value")
	}
	if cacheControl := resp.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("Cache-Control = %q, want %q", cacheControl, "no-store")
	}
}

func TestDemoInfoHidesAccountsFromRemoteClients(t *testing.T) {
	t.Setenv("SCENIC_GUIDE_DEMO_MODE", "true")
	t.Setenv("SCENIC_GUIDE_DEMO_PASSWORD", "ScenicDemo123456")

	_, body := requestDemoInfo(t, "192.0.2.10:12345", "127.0.0.1:8080")
	if body.Data.Enabled {
		t.Fatal("demo info unexpectedly enabled for remote client")
	}
	if len(body.Data.Accounts) != 0 {
		t.Fatalf("accounts = %d, want 0", len(body.Data.Accounts))
	}
}

func TestDemoInfoHidesAccountsBehindNonLocalHost(t *testing.T) {
	t.Setenv("SCENIC_GUIDE_DEMO_MODE", "true")
	t.Setenv("SCENIC_GUIDE_DEMO_PASSWORD", "ScenicDemo123456")

	_, body := requestDemoInfo(t, "127.0.0.1:12345", "demo.example.com")
	if body.Data.Enabled || len(body.Data.Accounts) != 0 {
		t.Fatal("demo info unexpectedly enabled behind a non-local host")
	}
}
