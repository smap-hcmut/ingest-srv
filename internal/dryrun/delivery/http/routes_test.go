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
		"success": {input: "/api", output: 3},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, _ := newTestHandler(t)
			r := gin.New()
			mw := middleware.New(middleware.Config{})

			h.RegisterRoutes(r.Group(tc.input), mw)

			routes := r.Routes()
			require.Len(t, routes, tc.output)
			requireRoute(t, routes, http.MethodPost, "/api/datasources/:id/dryrun")
			requireRoute(t, routes, http.MethodGet, "/api/datasources/:id/dryrun/latest")
			requireRoute(t, routes, http.MethodGet, "/api/datasources/:id/dryrun/history")
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
