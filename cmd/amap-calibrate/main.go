package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/scenic-guide/internal/geolocation"
)

func main() {
	file := flag.String("file", "./configs/scenic_spot_coordinates.json", "坐标校准文件")
	flag.Parse()

	apiKey := os.Getenv("AMAP_API_KEY")
	securityCode := os.Getenv("AMAP_SECURITY_CODE")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 20 * time.Second}
	if err := geolocation.RefreshCalibration(
		ctx,
		client,
		geolocation.AMapGeocodeEndpoint,
		apiKey,
		securityCode,
		*file,
	); err != nil {
		fmt.Fprintf(os.Stderr, "坐标校准失败，原文件未覆盖: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("坐标候选已写入 %s，请人工核对后填写 verified_at 并将 verified 设为 true\n", *file)
}
