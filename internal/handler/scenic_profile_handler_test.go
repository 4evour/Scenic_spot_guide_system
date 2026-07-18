package handler

import (
	"testing"

	"github.com/scenic-guide/internal/config"
)

func TestScenicProfileRoutesExposeSourceMetadata(t *testing.T) {
	handler := NewScenicProfileHandler(&config.ScenicProfile{
		Routes: []config.RouteConfig{{
			Name:             "观光车参考路线",
			RouteType:        "sightseeing_bus",
			Source:           "第三方转载路线",
			SourceURL:        "https://example.com/route",
			Confidence:       0.65,
			OfficialVerified: false,
		}},
	})

	routes := handler.GetRoutes()
	if len(routes) != 1 {
		t.Fatalf("route count = %d, want 1", len(routes))
	}
	route := routes[0]
	if route["route_type"] != "sightseeing_bus" || route["source"] != "第三方转载路线" {
		t.Fatalf("route source metadata = %+v", route)
	}
	if route["source_url"] != "https://example.com/route" || route["confidence"] != 0.65 {
		t.Fatalf("route source details = %+v", route)
	}
	if verified, ok := route["official_verified"].(bool); !ok || verified {
		t.Fatalf("official_verified = %#v, want false", route["official_verified"])
	}
}
