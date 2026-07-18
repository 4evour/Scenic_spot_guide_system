package geolocation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGeocodeResponse(t *testing.T) {
	data := []byte(`{"status":"1","info":"OK","count":"1","geocodes":[{"formatted_address":"江苏省无锡市滨湖区九龙灌浴","location":"120.100925,31.425920"}]}`)

	result, err := ParseGeocodeResponse(data, "九龙灌浴")
	if err != nil {
		t.Fatalf("ParseGeocodeResponse returned error: %v", err)
	}
	if result.Longitude != 120.100925 || result.Latitude != 31.425920 {
		t.Fatalf("coordinates = (%f, %f)", result.Longitude, result.Latitude)
	}
	if result.ReturnedAddress != "江苏省无锡市滨湖区九龙灌浴" {
		t.Fatalf("returned address = %q", result.ReturnedAddress)
	}
}

func TestParseGeocodeResponseRejectsUnsafeResults(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "empty", data: `{"status":"1","info":"OK","count":"0","geocodes":[]}`},
		{name: "out of bounds", data: `{"status":"1","info":"OK","count":"1","geocodes":[{"formatted_address":"九龙灌浴","location":"220.1,31.4"}]}`},
		{name: "uncertain name", data: `{"status":"1","info":"OK","count":"1","geocodes":[{"formatted_address":"江苏省无锡市滨湖区灵山胜境","location":"120.1,31.4"}]}`},
		{name: "api failure", data: `{"status":"0","info":"INVALID_USER_KEY","count":"0","geocodes":[]}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseGeocodeResponse([]byte(test.data), "九龙灌浴"); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestRefreshCalibrationDoesNotOverwriteFileWhenAnyQueryFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "coordinates.json")
	original := []byte(`{"version":1,"spots":[{"name":"九龙灌浴","query_address":"九龙灌浴 无锡灵山胜境","returned_address":"旧地址","longitude":120.099984,"latitude":31.424601,"coordinate_system":"GCJ-02","source":"amap-geocode-v3","verified_at":"2026-07-13T14:00:00+08:00","verified":true},{"name":"文创驿站","query_address":"文创驿站 无锡灵山胜境","returned_address":"旧地址","longitude":120.103651,"latitude":31.420196,"coordinate_system":"GCJ-02","source":"amap-geocode-v3","verified_at":"","verified":false}]}`)
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatalf("write original calibration: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "test-key" {
			t.Errorf("missing API key")
		}
		if r.URL.Query().Get("_jscode") != "test-security-code" {
			t.Errorf("missing optional security code")
		}
		if strings.Contains(r.URL.Query().Get("address"), "文创驿站") {
			_, _ = w.Write([]byte(`{"status":"1","info":"OK","count":"0","geocodes":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"1","info":"OK","count":"1","geocodes":[{"formatted_address":"江苏省无锡市滨湖区九龙灌浴","location":"120.100925,31.425920"}]}`))
	}))
	defer server.Close()

	err := RefreshCalibration(context.Background(), server.Client(), server.URL, "test-key", "test-security-code", path)
	if err == nil {
		t.Fatal("expected calibration failure")
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read calibration after failure: %v", readErr)
	}
	if string(after) != string(original) {
		t.Fatal("calibration file was overwritten after a partial failure")
	}
}

func TestRefreshCalibrationWritesCompleteUnverifiedCandidates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "coordinates.json")
	original := []byte(`{"version":1,"spots":[{"name":"九龙灌浴","query_address":"九龙灌浴 无锡灵山胜境","returned_address":"旧地址","longitude":120.099984,"latitude":31.424601,"coordinate_system":"GCJ-02","source":"amap-geocode-v3","verified_at":"2026-07-13T14:00:00+08:00","verified":true}]}`)
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatalf("write original calibration: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"1","info":"OK","count":"1","geocodes":[{"formatted_address":"江苏省无锡市滨湖区九龙灌浴","location":"120.100925,31.425920"}]}`))
	}))
	defer server.Close()

	if err := RefreshCalibration(context.Background(), server.Client(), server.URL, "test-key", "", path); err != nil {
		t.Fatalf("RefreshCalibration returned error: %v", err)
	}
	calibration, err := LoadCalibration(path)
	if err != nil {
		t.Fatalf("LoadCalibration returned error: %v", err)
	}
	spot := calibration.Spots[0]
	if spot.Longitude != 120.100925 || spot.Latitude != 31.425920 {
		t.Fatalf("coordinates = (%f, %f)", spot.Longitude, spot.Latitude)
	}
	if spot.Verified || spot.VerifiedAt != "" {
		t.Fatalf("new candidate must require manual verification: verified=%t verified_at=%q", spot.Verified, spot.VerifiedAt)
	}
}
