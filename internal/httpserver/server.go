package httpserver

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server wraps the HTTP server setup.
type Server struct {
	httpServer *http.Server
	logger     *log.Logger
	db         *pgxpool.Pool
}

// New builds a Server with basic routes.
func New(addr string, logger *log.Logger, db *pgxpool.Pool, deps Deps, fileURLHost string) (*Server, error) {
	router, err := buildRouter(logger, db, deps, fileURLHost)
	if err != nil {
		return nil, err
	}

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &Server{
		httpServer: httpSrv,
		logger:     logger,
		db:         db,
	}, nil
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func readyHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "reason": "db not configured"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "reason": "db not reachable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
