package geolocation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const AMapGeocodeEndpoint = "https://restapi.amap.com/v3/geocode/geo"

type CalibrationFile struct {
	Version int               `json:"version"`
	Spots   []SpotCalibration `json:"spots"`
}

type SpotCalibration struct {
	Name             string  `json:"name"`
	QueryAddress     string  `json:"query_address"`
	ReturnedAddress  string  `json:"returned_address"`
	Longitude        float64 `json:"longitude"`
	Latitude         float64 `json:"latitude"`
	CoordinateSystem string  `json:"coordinate_system"`
	Source           string  `json:"source"`
	AMapCandidate    bool    `json:"amap_candidate"`
	VerifiedAt       string  `json:"verified_at"`
	Verified         bool    `json:"verified"`
	GeofenceEnabled  bool    `json:"geofence_enabled"`
}

type GeocodeResult struct {
	ReturnedAddress string
	Longitude       float64
	Latitude        float64
}

type amapGeocodeResponse struct {
	Status   string `json:"status"`
	Info     string `json:"info"`
	Count    string `json:"count"`
	Geocodes []struct {
		FormattedAddress string `json:"formatted_address"`
		Location         string `json:"location"`
	} `json:"geocodes"`
}

func LoadCalibration(path string) (CalibrationFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CalibrationFile{}, fmt.Errorf("read coordinate calibration: %w", err)
	}
	var calibration CalibrationFile
	if err := json.Unmarshal(data, &calibration); err != nil {
		return CalibrationFile{}, fmt.Errorf("parse coordinate calibration: %w", err)
	}
	if calibration.Version != 1 || len(calibration.Spots) == 0 {
		return CalibrationFile{}, fmt.Errorf("coordinate calibration must contain version 1 and at least one spot")
	}
	seen := make(map[string]struct{}, len(calibration.Spots))
	for _, spot := range calibration.Spots {
		if _, ok := seen[spot.Name]; ok {
			return CalibrationFile{}, fmt.Errorf("duplicate calibrated spot %q", spot.Name)
		}
		seen[spot.Name] = struct{}{}
		if err := validateCalibrationSpot(spot); err != nil {
			return CalibrationFile{}, err
		}
	}
	return calibration, nil
}

func ParseGeocodeResponse(data []byte, expectedName string) (GeocodeResult, error) {
	var response amapGeocodeResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return GeocodeResult{}, fmt.Errorf("parse AMap response: %w", err)
	}
	if response.Status != "1" || len(response.Geocodes) == 0 {
		return GeocodeResult{}, fmt.Errorf("AMap geocode failed: %s", response.Info)
	}
	result := response.Geocodes[0]
	if !strings.Contains(result.FormattedAddress, expectedName) {
		return GeocodeResult{}, fmt.Errorf("AMap result does not clearly match %q", expectedName)
	}
	parts := strings.Split(result.Location, ",")
	if len(parts) != 2 {
		return GeocodeResult{}, fmt.Errorf("AMap result for %q has invalid location", expectedName)
	}
	lng, lngErr := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lat, latErr := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if lngErr != nil || latErr != nil || !IsValidCoordinate(lng, lat) {
		return GeocodeResult{}, fmt.Errorf("AMap result for %q has out-of-range coordinates", expectedName)
	}
	return GeocodeResult{ReturnedAddress: result.FormattedAddress, Longitude: lng, Latitude: lat}, nil
}

func IsValidCoordinate(longitude, latitude float64) bool {
	return longitude != 0 && latitude != 0 && longitude >= -180 && longitude <= 180 && latitude >= -90 && latitude <= 90
}

func RefreshCalibration(ctx context.Context, client *http.Client, endpoint, apiKey, securityCode, path string) error {
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("AMAP_API_KEY is required")
	}
	calibration, err := LoadCalibration(path)
	if err != nil {
		return err
	}
	updated := calibration
	for index, spot := range calibration.Spots {
		result, queryErr := queryAMap(ctx, client, endpoint, apiKey, securityCode, spot.Name, spot.QueryAddress)
		if queryErr != nil {
			return queryErr
		}
		updated.Spots[index].ReturnedAddress = result.ReturnedAddress
		updated.Spots[index].Longitude = result.Longitude
		updated.Spots[index].Latitude = result.Latitude
		updated.Spots[index].CoordinateSystem = "GCJ-02"
		updated.Spots[index].Source = "amap-geocode-v3"
		updated.Spots[index].AMapCandidate = true
		updated.Spots[index].Verified = false
		updated.Spots[index].VerifiedAt = ""
		updated.Spots[index].GeofenceEnabled = false
	}
	return writeCalibrationAtomically(path, updated)
}

func queryAMap(ctx context.Context, client *http.Client, endpoint, apiKey, securityCode, expectedName, address string) (GeocodeResult, error) {
	requestURL, err := url.Parse(endpoint)
	if err != nil {
		return GeocodeResult{}, fmt.Errorf("invalid AMap endpoint")
	}
	query := requestURL.Query()
	query.Set("key", apiKey)
	query.Set("address", address)
	if securityCode != "" {
		query.Set("_jscode", securityCode)
	}
	requestURL.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return GeocodeResult{}, fmt.Errorf("create AMap request")
	}
	response, err := client.Do(request)
	if err != nil {
		return GeocodeResult{}, fmt.Errorf("AMap geocode request failed for %q", expectedName)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return GeocodeResult{}, fmt.Errorf("AMap geocode returned HTTP %d for %q", response.StatusCode, expectedName)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return GeocodeResult{}, fmt.Errorf("read AMap response for %q", expectedName)
	}
	return ParseGeocodeResponse(body, expectedName)
}

func validateCalibrationSpot(spot SpotCalibration) error {
	if strings.TrimSpace(spot.Name) == "" || strings.TrimSpace(spot.QueryAddress) == "" {
		return fmt.Errorf("coordinate calibration contains a spot without name or query address")
	}
	if strings.TrimSpace(spot.ReturnedAddress) == "" || spot.CoordinateSystem != "GCJ-02" || strings.TrimSpace(spot.Source) == "" {
		return fmt.Errorf("coordinate calibration for %q is missing returned address, coordinate system, or source", spot.Name)
	}
	if !IsValidCoordinate(spot.Longitude, spot.Latitude) {
		return fmt.Errorf("coordinate calibration for %q has out-of-range coordinates", spot.Name)
	}
	if spot.Verified && strings.TrimSpace(spot.VerifiedAt) == "" {
		return fmt.Errorf("verified coordinate calibration for %q is missing verified_at", spot.Name)
	}
	return nil
}

func writeCalibrationAtomically(path string, calibration CalibrationFile) error {
	data, err := json.MarshalIndent(calibration, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize coordinate calibration: %w", err)
	}
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".coordinates-*.json")
	if err != nil {
		return fmt.Errorf("create temporary coordinate calibration: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary coordinate calibration: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary coordinate calibration: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace coordinate calibration: %w", err)
	}
	return nil
}
