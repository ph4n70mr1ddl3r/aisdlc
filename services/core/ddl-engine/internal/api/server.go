// Package api wires ddl-engine's HTTP surface: dry-run DDL preview, apply for
// one entity or a whole tenant, and the migration ledger. Mirrors metadata-api's
// middleware conventions (CORS, request IDs, tenant header resolution).
package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ph4n70mr1ddl3r/aisdlc/services/core/ddl-engine/internal/ddl"
)

var corsOrigin string

const (
	maxBodyBytes = 1 << 20
	ctxTenant    = "tenantID"
)

func init() {
	corsOrigin = os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "http://localhost:3000"
	}
}

// NewRouter builds the gin engine with /healthz and /v1 routes.
func NewRouter(engine *ddl.Engine) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger(), correlationID, corsMiddleware, maxBodyMiddleware)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/v1")
	{
		v1.GET("/entities/:id/ddl", entityDDL(engine))
		v1.POST("/entities/:id/apply", applyEntity(engine))
		v1.POST("/apply", applyTenant(engine))
		v1.GET("/migrations", listMigrations(engine))
	}
	return r
}

// entityDDL is a dry run: returns the statements that would be applied.
func entityDDL(engine *ddl.Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		plan, err := engine.Plan(c.Request.Context(), c.Param("id"))
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"entity":     plan.Entity,
			"statements": plan.Statements,
			"count":      len(plan.Statements),
		})
	}
}

// applyEntity applies DDL for a single entity by id.
func applyEntity(engine *ddl.Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := engine.Apply(c.Request.Context(), c.Param("id"))
		if err != nil {
			fail(c, err, res) // res may carry partial progress
			return
		}
		c.JSON(http.StatusOK, res)
	}
}

// applyTenant applies DDL for every entity owned by the resolved tenant.
func applyTenant(engine *ddl.Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		t := tenantOf(c)
		if t == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header (or ?tenant_id=) is required"})
			return
		}
		results, err := engine.ApplyAll(c.Request.Context(), t)
		out := gin.H{"results": results, "count": len(results)}
		if err != nil {
			out["error"] = err.Error()
			c.JSON(http.StatusOK, out) // partial; some entities may have applied
			return
		}
		c.JSON(http.StatusOK, out)
	}
}

// listMigrations returns the recorded DDL ledger, filterable by tenant/entity.
func listMigrations(engine *ddl.Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		ms, err := engine.Migrations(c.Request.Context(), tenantOf(c), c.Query("entity_id"))
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": ms, "count": len(ms)})
	}
}

// ── middleware ───────────────────────────────────────────────

func correlationID(c *gin.Context) {
	id := c.GetHeader("X-Request-ID")
	if id == "" {
		var b [16]byte
		if _, err := rand.Read(b[:]); err != nil {
			id = fmt.Sprintf("%016x", time.Now().UnixNano())
		} else {
			id = hex.EncodeToString(b[:])
		}
	}
	c.Set("requestID", id)
	c.Header("X-Request-ID", id)
	c.Next()
}

func corsMiddleware(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", corsOrigin)
	c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type, X-Tenant-ID")
	if c.Request.Method == http.MethodOptions {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	c.Next()
}

func maxBodyMiddleware(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	c.Next()
}

func tenantOf(c *gin.Context) string {
	t := strings.TrimSpace(c.GetHeader("X-Tenant-ID"))
	if t == "" {
		t = strings.TrimSpace(c.Query("tenant_id"))
	}
	return t
}

func fail(c *gin.Context, err error, partial ...any) {
	reqID, _ := c.Get("requestID")
	switch {
	case errors.Is(err, ddl.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "entity not found"})
	case errors.Is(err, ddl.ErrUnsupported):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	default:
		log.Printf("request=%v internal error: %v", reqID, err)
		if len(partial) > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error", "partial": partial[0]})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
