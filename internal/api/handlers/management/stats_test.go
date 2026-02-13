package management

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/storage/postgres"
	"golang.org/x/crypto/bcrypt"
)

type fakePostgresPlugin struct {
	active bool
	pool   *postgres.Pool
}

func (f *fakePostgresPlugin) IsActive() bool { return f.active }

func (f *fakePostgresPlugin) Pool() *postgres.Pool { return f.pool }

func TestGetVendorErrorLogs_ParsesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	var gotOpts postgres.VendorErrorLogListOptions
	origQuery := queryVendorErrorLogs
	queryVendorErrorLogs = func(ctx context.Context, pool *pgxpool.Pool, opts postgres.VendorErrorLogListOptions) (*postgres.VendorErrorLogListResult, error) {
		gotOpts = opts
		return &postgres.VendorErrorLogListResult{
			Entries: []postgres.VendorErrorLogEntry{{Provider: opts.Provider}},
			Total:   1,
			Page:    opts.Page,
			Limit:   opts.Limit,
			TimeRange: postgres.TimeRange{
				Start: opts.StartTime,
				End:   opts.EndTime,
			},
			Provider: opts.Provider,
		}, nil
	}
	t.Cleanup(func() { queryVendorErrorLogs = origQuery })

	h := &Handler{postgresPlugin: &fakePostgresPlugin{active: true, pool: &postgres.Pool{}}}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/vendor-error-logs?provider=acme&page=2&limit=10&start=2025-01-01T00:00:00Z&end=2025-01-02T00:00:00Z", nil)

	h.GetVendorErrorLogs(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if gotOpts.Provider != "acme" {
		t.Fatalf("expected provider 'acme', got %q", gotOpts.Provider)
	}
	if gotOpts.Page != 2 {
		t.Fatalf("expected page 2, got %d", gotOpts.Page)
	}
	if gotOpts.Limit != 10 {
		t.Fatalf("expected limit 10, got %d", gotOpts.Limit)
	}
	if gotOpts.StartTime == nil || !gotOpts.StartTime.Equal(start) {
		t.Fatalf("expected start time %v, got %v", start, gotOpts.StartTime)
	}
	if gotOpts.EndTime == nil || !gotOpts.EndTime.Equal(end) {
		t.Fatalf("expected end time %v, got %v", end, gotOpts.EndTime)
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json response: %v", err)
	}
}

func TestManagementMiddleware_UnauthorizedWithoutKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secretHash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash secret: %v", err)
	}

	cfg := &config.Config{}
	cfg.RemoteManagement.SecretKey = string(secretHash)

	h := &Handler{cfg: cfg}
	called := false
	engine := gin.New()
	engine.Use(h.Middleware())
	engine.GET("/vendor-error-logs", func(c *gin.Context) {
		called = true
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/vendor-error-logs", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
	if called {
		t.Fatalf("expected handler not to be called when unauthorized")
	}
}
