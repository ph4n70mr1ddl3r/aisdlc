package api

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ph4n70mr1ddl3r/aisdlc/services/core/metadata-api/internal/schema"
	"github.com/ph4n70mr1ddl3r/aisdlc/services/core/metadata-api/internal/store"
)

const (
	ctxTenant    = "tenantID"
	maxBodyBytes = 1 << 20 // 1 MB
)

// NewRouter builds the gin engine with /healthz and the /v1 CRUD routes.
func NewRouter(s *store.Store) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger(), resolveTenant, corsMiddleware, maxBodyMiddleware)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	a := &API{store: s, byName: map[string]*schema.Resource{}}
	for _, res := range Resources() {
		a.byName[res.Name] = res
	}
	v1 := r.Group("/v1")
	for _, res := range Resources() {
		res := res // capture
		v1.GET("/"+res.Name, a.list(res))
		v1.POST("/"+res.Name, a.create(res))
		v1.GET("/"+res.Name+"/:id", a.get(res))
		v1.PATCH("/"+res.Name+"/:id", a.update(res))
		v1.DELETE("/"+res.Name+"/:id", a.delete(res))
	}
	return r
}

// API holds the store and the resource registry.
type API struct {
	store *store.Store
	byName map[string]*schema.Resource
}

// resolveTenant reads X-Tenant-ID (or ?tenant_id=) into the request context.
// In M1 there is no auth; identity (M2) will set this from the session.
func maxBodyMiddleware(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	c.Next()
}

func resolveTenant(c *gin.Context) {
	t := strings.TrimSpace(c.GetHeader("X-Tenant-ID"))
	if t == "" {
		t = strings.TrimSpace(c.Query("tenant_id"))
	}
	if t != "" {
		c.Set(ctxTenant, t)
	}
	c.Next()
}

// corsMiddleware reads CORS_ORIGIN (default *) for development. Lock to a
// specific origin (e.g. http://localhost:3000) in production.
func corsMiddleware(c *gin.Context) {
	origin := os.Getenv("CORS_ORIGIN")
	if origin == "" {
		origin = "*"
	}
	c.Header("Access-Control-Allow-Origin", origin)
	c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID")
	if c.Request.Method == http.MethodOptions {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	c.Next()
}

// tenantOf returns the resolved tenant id (may be "").
func tenantOf(c *gin.Context) string {
	v, _ := c.Get(ctxTenant)
	s, _ := v.(string)
	return s
}

func (a *API) list(r *schema.Resource) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := store.ListQuery{
			TenantID: tenantOf(c),
			Search:   c.Query("q"),
			Order:    c.Query("order"),
			Limit:    atoi(c.Query("limit")),
			Offset:   atoi(c.Query("offset")),
			Filters:  queryFilters(c, r),
		}
		items, total, err := a.store.List(c.Request.Context(), r, q)
		if err != nil {
			a.fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"data":   items,
			"total":  total,
			"limit":  q.Limit,
			"offset": q.Offset,
		})
	}
}

func (a *API) get(r *schema.Resource) gin.HandlerFunc {
	return func(c *gin.Context) {
		out, err := a.store.Get(c.Request.Context(), r, c.Param("id"), tenantOf(c))
		if err != nil {
			a.fail(c, err)
			return
		}
		c.JSON(http.StatusOK, out)
	}
}

func (a *API) create(r *schema.Resource) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body map[string]any
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		out, err := a.store.Create(c.Request.Context(), r, body, tenantOf(c))
		if err != nil {
			a.fail(c, err)
			return
		}
		c.JSON(http.StatusCreated, out)
	}
}

func (a *API) update(r *schema.Resource) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body map[string]any
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		out, err := a.store.Update(c.Request.Context(), r, c.Param("id"), body, tenantOf(c))
		if err != nil {
			a.fail(c, err)
			return
		}
		c.JSON(http.StatusOK, out)
	}
}

func (a *API) delete(r *schema.Resource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := a.store.Delete(c.Request.Context(), r, c.Param("id"), tenantOf(c)); err != nil {
			a.fail(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// queryFilters collects ?col=value params that match a real column (excluding
// reserved controls), so callers can filter e.g. ?app_id=…&entity_id=….
func queryFilters(c *gin.Context, r *schema.Resource) map[string]string {
	reserved := map[string]bool{
		"q": true, "limit": true, "offset": true, "order": true, "tenant_id": true,
	}
	f := map[string]string{}
	for k, vs := range c.Request.URL.Query() {
		if reserved[k] || len(vs) == 0 {
			continue
		}
		if r.Column(k) != nil {
			f[k] = vs[0]
		}
	}
	return f
}

// fail maps a store error to an HTTP status + body.
func (a *API) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, store.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "conflict — a row with this unique key already exists"})
	case errors.Is(err, store.ErrFKViolation):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "referenced row does not exist"})
	case errors.Is(err, store.ErrTenantReq):
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header (or ?tenant_id=) is required for this resource"})
	case errors.Is(err, store.ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func atoi(s string) int {
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0
	}
	return n
}
