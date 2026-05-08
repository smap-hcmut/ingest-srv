package http

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/middleware"
	"github.com/stretchr/testify/require"
)

func TestRegisterRoutes(t *testing.T) {
	tcs := map[string]struct {
		input  string
		mock   struct{}
		output int
		err    error
	}{
		"success": {input: "/api", output: 15},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, _ := newTestHandler(t)
			r := gin.New()
			mw := middleware.New(middleware.Config{})

			h.RegisterRoutes(r.Group(tc.input), mw)

			routes := r.Routes()
			require.Len(t, routes, tc.output)
			requireRoute(t, routes, http.MethodPost, "/api/datasources")
			requireRoute(t, routes, http.MethodGet, "/api/datasources")
			requireRoute(t, routes, http.MethodGet, "/api/datasources/:id")
			requireRoute(t, routes, http.MethodPut, "/api/datasources/:id")
			requireRoute(t, routes, http.MethodPost, "/api/datasources/:id/archive")
			requireRoute(t, routes, http.MethodDelete, "/api/datasources/:id")
			requireRoute(t, routes, http.MethodPost, "/api/datasources/:id/targets/keywords")
			requireRoute(t, routes, http.MethodPost, "/api/datasources/:id/targets/profiles")
			requireRoute(t, routes, http.MethodPost, "/api/datasources/:id/targets/posts")
			requireRoute(t, routes, http.MethodGet, "/api/datasources/:id/targets")
			requireRoute(t, routes, http.MethodGet, "/api/datasources/:id/targets/:target_id")
			requireRoute(t, routes, http.MethodPut, "/api/datasources/:id/targets/:target_id")
			requireRoute(t, routes, http.MethodPost, "/api/datasources/:id/targets/:target_id/activate")
			requireRoute(t, routes, http.MethodPost, "/api/datasources/:id/targets/:target_id/deactivate")
			requireRoute(t, routes, http.MethodDelete, "/api/datasources/:id/targets/:target_id")
		})
	}
}

func TestRegisterInternalRoutes(t *testing.T) {
	tcs := map[string]struct {
		input  string
		mock   struct{}
		output int
		err    error
	}{
		"success": {input: "/internal", output: 6},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, _ := newTestHandler(t)
			r := gin.New()
			mw := middleware.New(middleware.Config{InternalKey: "internal-key"})

			h.RegisterInternalRoutes(r.Group(tc.input), mw)

			routes := r.Routes()
			require.Len(t, routes, tc.output)
			requireRoute(t, routes, http.MethodPut, "/internal/datasources/:id/crawl-mode")
			requireRoute(t, routes, http.MethodGet, "/internal/projects/:project_id/activation-readiness")
			requireRoute(t, routes, http.MethodPost, "/internal/projects/:project_id/activate")
			requireRoute(t, routes, http.MethodPost, "/internal/projects/:project_id/pause")
			requireRoute(t, routes, http.MethodPost, "/internal/projects/:project_id/resume")
			requireRoute(t, routes, http.MethodPost, "/internal/projects/:project_id/crawl-mode")
		})
	}
}

func requireRoute(t *testing.T, routes gin.RoutesInfo, method string, path string) {
	t.Helper()
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return
		}
	}
	require.Failf(t, "route not found", "%s %s", method, path)
}
