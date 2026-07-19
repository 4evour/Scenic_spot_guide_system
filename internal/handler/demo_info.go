package handler

import (
	"net"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/pkg"
)

type demoAccount struct {
	Role     string `json:"role"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type demoInfoResponse struct {
	Enabled  bool          `json:"enabled"`
	Accounts []demoAccount `json:"accounts,omitempty"`
}

func demoInfo(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")

	password := strings.TrimSpace(os.Getenv("SCENIC_GUIDE_DEMO_PASSWORD"))
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("SCENIC_GUIDE_DEMO_MODE")), "true") ||
		password == "" || !isLoopbackRemoteAddr(c.Request.RemoteAddr) || !isLoopbackHost(c.Request.Host) {
		pkg.Success(c, demoInfoResponse{Enabled: false})
		return
	}

	pkg.Success(c, demoInfoResponse{
		Enabled: true,
		Accounts: []demoAccount{
			{Role: "visitor", Username: "visitor", Password: password},
			{Role: "admin", Username: "admin", Password: password},
		},
	})
}

func isLoopbackHost(hostPort string) bool {
	host := strings.TrimSpace(hostPort)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isLoopbackRemoteAddr(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}
