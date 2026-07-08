package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/config"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/pkg"
)

type fakeVisitorQueryService struct{}

func (fakeVisitorQueryService) CreateQuery(_ *model.VisitorQuery) error {
	return nil
}

func (fakeVisitorQueryService) GetQueryByID(_ uint) (*model.VisitorQuery, error) {
	return &model.VisitorQuery{ID: 99, Query: "wrong handler"}, nil
}

func (fakeVisitorQueryService) GetAllQueries() ([]model.VisitorQuery, error) {
	return []model.VisitorQuery{{ID: 1, Query: "all"}}, nil
}

func (fakeVisitorQueryService) GetUnansweredQueries() ([]model.VisitorQuery, error) {
	return []model.VisitorQuery{{ID: 2, Query: "unanswered"}}, nil
}

func (fakeVisitorQueryService) UpdateQuery(_ *model.VisitorQuery) error {
	return nil
}

func (fakeVisitorQueryService) DeleteQuery(_ uint) error {
	return nil
}

func TestVisitorQueryRoutesRegisterUnansweredList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	queryHandler := NewVisitorQueryHandler(nil)

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("register visitor query routes: %v", recovered)
		}
	}()

	queryHandler.Routes(router.Group("/api/v1"))
}

func TestVisitorQueryUnansweredRouteUsesAdminListHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if err := pkg.InitJWT(&config.SecurityConfig{JWTSecret: "0123456789abcdef0123456789abcdef"}); err != nil {
		t.Fatalf("InitJWT: %v", err)
	}

	router := gin.New()
	queryHandler := NewVisitorQueryHandler(fakeVisitorQueryService{})
	queryHandler.Routes(router.Group("/api/v1"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queries/unanswered", nil)
	req.RemoteAddr = "192.0.2.80:1234"
	req.Header.Set("Authorization", authHeaderFor(t, 1, "admin", "admin"))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "unanswered") {
		t.Fatalf("unanswered route did not use admin list handler, body=%s", resp.Body.String())
	}
}
