package httpserver

import (
	"context"
	"log"
	"net/http"

	"commercetools-replica/internal/domain"
	projectrepo "commercetools-replica/internal/repository/project"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Deps struct {
	ProjectRepo projectrepo.Repository
}

func buildRouter(logger *log.Logger, db *pgxpool.Pool, deps Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.LoggerWithWriter(logger.Writer()), gin.Recovery())

	router.GET("/healthz", healthHandler)
	router.GET("/readyz", readyHandler(db))

	projectGroup := router.Group("/projects/:projectKey", projectMiddleware(deps.ProjectRepo))
	{
		projectGroup.GET("/products", func(c *gin.Context) {
			// Placeholder: will list products for project
			c.JSON(http.StatusOK, gin.H{"message": "products endpoint placeholder"})
		})
		projectGroup.GET("/products/:id", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "product detail placeholder"})
		})
		projectGroup.POST("/carts", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "cart create placeholder"})
		})
		projectGroup.GET("/carts/:id", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "cart get placeholder"})
		})
	}

	return router
}

type ctxKey string

const projectCtxKey ctxKey = "project"

func projectMiddleware(repo projectrepo.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("projectKey")
		if key == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "projectKey required"})
			c.Abort()
			return
		}
		project, err := repo.GetByKey(c.Request.Context(), key)
		if err != nil {
			if err == domain.ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
				c.Abort()
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "project lookup failed"})
			c.Abort()
			return
		}
		ctx := context.WithValue(c.Request.Context(), projectCtxKey, project)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
