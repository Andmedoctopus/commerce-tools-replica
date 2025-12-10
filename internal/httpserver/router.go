package httpserver

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// buildRouter wires routes for the API.
func buildRouter(logger *log.Logger, db *pgxpool.Pool) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.LoggerWithWriter(logger.Writer()), gin.Recovery())

	router.GET("/healthz", healthHandler)
	router.GET("/readyz", readyHandler(db))

	// TODO: add project-scoped routes for auth/products/carts.

	return router
}
