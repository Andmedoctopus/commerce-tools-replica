package httpserver

import (
	"context"
	"errors"
	"log"
	"net/http"

	"commercetools-replica/internal/domain"
	projectrepo "commercetools-replica/internal/repository/project"
	cartsvc "commercetools-replica/internal/service/cart"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type productService interface {
	List(ctx context.Context, projectID string) ([]domain.Product, error)
	Get(ctx context.Context, projectID, id string) (*domain.Product, error)
}

type cartService interface {
	Create(ctx context.Context, projectID string, in cartsvc.CreateInput) (*domain.Cart, error)
	Get(ctx context.Context, projectID, id string) (*domain.Cart, error)
}

type Deps struct {
	ProjectRepo projectrepo.Repository
	ProductSvc  productService
	CartSvc     cartService
}

func buildRouter(logger *log.Logger, db *pgxpool.Pool, deps Deps) (*gin.Engine, error) {
	if deps.ProjectRepo == nil {
		return nil, errors.New("ProjectRepo is required")
	}
	if deps.ProductSvc == nil {
		return nil, errors.New("ProductSvc is required")
	}
	if deps.CartSvc == nil {
		return nil, errors.New("CartSvc is required")
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.LoggerWithWriter(logger.Writer()), gin.Recovery())

	router.GET("/healthz", healthHandler)
	router.GET("/readyz", readyHandler(db))

	projectGroup := router.Group("/projects/:projectKey", projectMiddleware(deps.ProjectRepo))
	{
		projectGroup.GET("/products", func(c *gin.Context) {
			project := mustProject(c)
			products, err := deps.ProductSvc.List(c.Request.Context(), project.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "list products failed"})
				return
			}
			c.JSON(http.StatusOK, products)
		})
		projectGroup.GET("/products/:id", func(c *gin.Context) {
			project := mustProject(c)
			id := c.Param("id")
			p, err := deps.ProductSvc.Get(c.Request.Context(), project.ID, id)
			if err != nil {
				if err == domain.ErrNotFound {
					c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "get product failed"})
				return
			}
			c.JSON(http.StatusOK, p)
		})
		projectGroup.POST("/carts", func(c *gin.Context) {
			project := mustProject(c)
			var req cartsvc.CreateInput
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
				return
			}
			cart, err := deps.CartSvc.Create(c.Request.Context(), project.ID, req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, cart)
		})
		projectGroup.GET("/carts/:id", func(c *gin.Context) {
			project := mustProject(c)
			id := c.Param("id")
			cart, err := deps.CartSvc.Get(c.Request.Context(), project.ID, id)
			if err != nil {
				if err == domain.ErrNotFound {
					c.JSON(http.StatusNotFound, gin.H{"error": "cart not found"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "get cart failed"})
				return
			}
			c.JSON(http.StatusOK, cart)
		})
	}

	return router, nil
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
		c.Set(string(projectCtxKey), project)
		c.Next()
	}
}

func currentProject(c *gin.Context) *domain.Project {
	if val, ok := c.Get(string(projectCtxKey)); ok {
		if p, ok := val.(*domain.Project); ok {
			return p
		}
	}
	if val := c.Request.Context().Value(projectCtxKey); val != nil {
		if p, ok := val.(*domain.Project); ok {
			return p
		}
	}
	return nil
}

func mustProject(c *gin.Context) *domain.Project {
	p := currentProject(c)
	if p == nil {
		panic("project missing in context")
	}
	return p
}
