package httpserver

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"commercetools-replica/internal/domain"
	projectrepo "commercetools-replica/internal/repository/project"
	cartsvc "commercetools-replica/internal/service/cart"
	"github.com/gin-contrib/cors"
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

type categoryService interface {
	List(ctx context.Context, projectID string) ([]domain.Category, error)
	Upsert(ctx context.Context, c domain.Category) (*domain.Category, error)
}

type Deps struct {
	ProjectRepo projectrepo.Repository
	ProductSvc  productService
	CartSvc     cartService
	CategorySvc categoryService
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
	if deps.CategorySvc == nil {
		return nil, errors.New("CategorySvc is required")
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// allow any localhost/127.x origin (any port)
			return strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1")
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "Accept"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
		AllowWildcard:    true,
	}))
	router.Use(requestLogger(logger))

	router.GET("/healthz", healthHandler)
	router.GET("/readyz", readyHandler(db))

	registerProjectRoutes := func(group *gin.RouterGroup) {
		group.GET("/products", func(c *gin.Context) {
			project := mustProject(c)
			products, err := deps.ProductSvc.List(c.Request.Context(), project.ID)
			if err != nil {
				logger.Printf("products list error project_id=%s error=%v", project.ID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "list products failed"})
				return
			}
			var resp []ctProduct
			for _, p := range products {
				resp = append(resp, toCTProduct(p))
			}
			c.JSON(http.StatusOK, resp)
		})
		group.GET("/products/:id", func(c *gin.Context) {
			project := mustProject(c)
			id := c.Param("id")
			p, err := deps.ProductSvc.Get(c.Request.Context(), project.ID, id)
			if err != nil {
				if err == domain.ErrNotFound {
					logger.Printf("product get not found project_id=%s id=%s", project.ID, id)
					c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
					return
				}
				logger.Printf("product get error project_id=%s id=%s error=%v", project.ID, id, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "get product failed"})
				return
			}
			c.JSON(http.StatusOK, toCTProduct(*p))
		})
		group.POST("/products/search", func(c *gin.Context) {
			project := mustProject(c)

			var req searchRequest
			if c.Request.Body != nil && c.Request.ContentLength != 0 {
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid search request"})
					return
				}
			}

			products, err := deps.ProductSvc.List(c.Request.Context(), project.ID)
			if err != nil {
				logger.Printf("products search error project_id=%s error=%v", project.ID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
				return
			}

			resp := buildSearchResponse(products, req)
			c.JSON(http.StatusOK, resp)
		})
		group.GET("/categories", func(c *gin.Context) {
			project := mustProject(c)
			cats, err := deps.CategorySvc.List(c.Request.Context(), project.ID)
			if err != nil {
				logger.Printf("categories list error project_id=%s error=%v", project.ID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "list categories failed"})
				return
			}
			limit, offset := parseLimitOffset(c.Query("limit"), c.Query("offset"))
			resp := buildCategoryList(cats, limit, offset)
			c.JSON(http.StatusOK, resp)
		})
		group.POST("/carts", func(c *gin.Context) {
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
		group.GET("/carts/:id", func(c *gin.Context) {
			project := mustProject(c)
			id := c.Param("id")
			cart, err := deps.CartSvc.Get(c.Request.Context(), project.ID, id)
			if err != nil {
				if err == domain.ErrNotFound {
					logger.Printf("cart get not found project_id=%s id=%s", project.ID, id)
					c.JSON(http.StatusNotFound, gin.H{"error": "cart not found"})
					return
				}
				logger.Printf("cart get error project_id=%s id=%s error=%v", project.ID, id, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "get cart failed"})
				return
			}
			c.JSON(http.StatusOK, cart)
		})
	}

	// commercetools-style prefix: /{projectKey}/...
	ctStyle := router.Group("/:projectKey", projectMiddleware(logger, deps.ProjectRepo))
	registerProjectRoutes(ctStyle)

	return router, nil
}

type ctxKey string

const projectCtxKey ctxKey = "project"

func projectMiddleware(logger *log.Logger, repo projectrepo.Repository) gin.HandlerFunc {
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
				logger.Printf("project middleware: key=%s not found", key)
				c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
				c.Abort()
				return
			}
			logger.Printf("project middleware: key=%s lookup error=%v", key, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "project lookup failed"})
			c.Abort()
			return
		}
		logger.Printf("project middleware: key=%s id=%s", project.Key, project.ID)
		ctx := context.WithValue(c.Request.Context(), projectCtxKey, project)
		c.Request = c.Request.WithContext(ctx)
		c.Set(string(projectCtxKey), project)
		c.Next()
	}
}

func requestLogger(logger *log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		proj := currentProject(c)
		pid := ""
		pkey := c.Param("projectKey")
		if proj != nil {
			pid = proj.ID
			pkey = proj.Key
		}

		logger.Printf("http %s %s status=%d dur=%s project_key=%s project_id=%s",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration.Truncate(time.Millisecond),
			pkey,
			pid,
		)
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
