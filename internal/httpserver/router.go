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
	anonymoussvc "commercetools-replica/internal/service/anonymous"
	cartsvc "commercetools-replica/internal/service/cart"
	customersvc "commercetools-replica/internal/service/customer"
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
	GetActive(ctx context.Context, projectID, customerID string) (*domain.Cart, error)
	Update(ctx context.Context, projectID, customerID, cartID string, in cartsvc.UpdateInput) (*domain.Cart, error)
	GetActiveAnonymous(ctx context.Context, projectID, anonymousID string) (*domain.Cart, error)
	UpdateAnonymous(ctx context.Context, projectID, anonymousID, cartID string, in cartsvc.UpdateInput) (*domain.Cart, error)
	AssignCustomerFromAnonymous(ctx context.Context, projectID, anonymousID, customerID string) (*domain.Cart, error)
	Delete(ctx context.Context, projectID, customerID, cartID string) (*domain.Cart, error)
	DeleteAnonymous(ctx context.Context, projectID, anonymousID, cartID string) (*domain.Cart, error)
}

type categoryService interface {
	List(ctx context.Context, projectID string) ([]domain.Category, error)
	Upsert(ctx context.Context, c domain.Category) (*domain.Category, error)
}

type customerService interface {
	Signup(ctx context.Context, projectID string, in customersvc.SignupInput) (*domain.Customer, error)
	Login(ctx context.Context, projectID, email, password string) (*domain.Customer, string, string, error)
	LookupByToken(ctx context.Context, projectID, token string) (*domain.Customer, error)
	AccessTTLSeconds() int
}

type anonymousService interface {
	Issue(ctx context.Context, projectID string) (string, string, string, error)
	LookupByToken(ctx context.Context, projectID, token string) (string, error)
	AccessTTLSeconds() int
}

type Deps struct {
	ProjectRepo  projectrepo.Repository
	ProductSvc   productService
	CartSvc      cartService
	CategorySvc  categoryService
	CustomerSvc  customerService
	AnonymousSvc anonymousService
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
	if deps.CustomerSvc == nil {
		return nil, errors.New("CustomerSvc is required")
	}
	if deps.AnonymousSvc == nil {
		return nil, errors.New("AnonymousSvc is required")
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
		group.POST("/me/signup", func(c *gin.Context) {
			project := mustProject(c)

			var req signupRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
				return
			}

			in := customersvc.SignupInput{
				Email:                  req.Email,
				Password:               req.Password,
				FirstName:              req.FirstName,
				LastName:               req.LastName,
				DateOfBirth:            req.DateOfBirth,
				DefaultShippingAddress: req.DefaultShippingAddress,
				DefaultBillingAddress:  req.DefaultBillingAddress,
			}
			for _, a := range req.Addresses {
				in.Addresses = append(in.Addresses, customersvc.AddressInput{
					FirstName:  a.FirstName,
					LastName:   a.LastName,
					Country:    a.Country,
					StreetName: a.StreetName,
					PostalCode: a.PostalCode,
					City:       a.City,
					Email:      a.Email,
					Department: a.Department,
				})
			}

			customer, err := deps.CustomerSvc.Signup(c.Request.Context(), project.ID, in)
			if err != nil {
				logger.Printf("customer signup error project_id=%s email=%s error=%v", project.ID, req.Email, err)
				status := http.StatusInternalServerError
				msg := "signup failed"
				switch {
				case errors.Is(err, domain.ErrAlreadyExists):
					status = http.StatusConflict
					msg = "customer already exists"
				default:
					if strings.Contains(strings.ToLower(err.Error()), "password") || strings.Contains(strings.ToLower(err.Error()), "email") {
						status = http.StatusBadRequest
						msg = err.Error()
					}
				}
				c.JSON(status, gin.H{"error": msg})
				return
			}

			if authHeader := c.GetHeader("Authorization"); authHeader != "" {
				token := extractBearerToken(authHeader)
				if token != "" {
					if anonymousID, err := deps.AnonymousSvc.LookupByToken(c.Request.Context(), project.ID, token); err == nil {
						if _, err := deps.CartSvc.AssignCustomerFromAnonymous(c.Request.Context(), project.ID, anonymousID, customer.ID); err != nil && !errors.Is(err, domain.ErrNotFound) {
							logger.Printf("signup cart transfer error project_id=%s customer_id=%s error=%v", project.ID, customer.ID, err)
						}
					}
				}
			}

			c.JSON(http.StatusCreated, customerResponse{Customer: toCTCustomer(*customer)})
		})
		group.GET("/me", func(c *gin.Context) {
			project := mustProject(c)
			customer, ok := authorizeCustomer(c, project, deps.CustomerSvc)
			if !ok {
				return
			}
			c.JSON(http.StatusOK, toCTCustomer(*customer))
		})
		group.POST("/me/login", func(c *gin.Context) {
			project := mustProject(c)

			var req loginRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid login request"})
				return
			}

			customer, _, _, err := deps.CustomerSvc.Login(c.Request.Context(), project.ID, req.Email, req.Password)
			if err != nil {
				status := http.StatusUnauthorized
				msg := "invalid credentials"
				if err != customersvc.ErrInvalidCredentials {
					status = http.StatusInternalServerError
					msg = "login failed"
				}
				c.JSON(status, gin.H{"error": msg})
				return
			}

			var cartResp *ctCart
			cart, err := deps.CartSvc.GetActive(c.Request.Context(), project.ID, customer.ID)
			if err != nil {
				if !errors.Is(err, domain.ErrNotFound) {
					logger.Printf("login cart lookup error project_id=%s customer_id=%s error=%v", project.ID, customer.ID, err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
					return
				}
			} else {
				ct := toCTCart(*cart, customer)
				cartResp = &ct
			}

			c.JSON(http.StatusOK, loginResponse{
				Customer: toCTCustomer(*customer),
				Cart:     cartResp,
			})
		})
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

			cats, err := deps.CategorySvc.List(c.Request.Context(), project.ID)
			if err != nil {
				// Don't fail the search entirely if categories lookup fails; just skip ID<->key mapping.
				logger.Printf("products search categories list error project_id=%s error=%v", project.ID, err)
				cats = nil
			}
			resp := buildSearchResponse(products, cats, req)
			c.JSON(http.StatusOK, resp)
		})
		group.GET("/product-discounts", func(c *gin.Context) {
			_ = mustProject(c)
			limit, offset := parseLimitOffset(c.Query("limit"), c.Query("offset"))
			c.JSON(http.StatusOK, buildProductDiscountList(limit, offset))
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
		group.POST("/me/carts", func(c *gin.Context) {
			project := mustProject(c)
			actor, ok := authorizeActor(c, project, deps.CustomerSvc, deps.AnonymousSvc)
			if !ok {
				return
			}
			var req cartsvc.CreateInput
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
				return
			}
			if actor.Customer != nil {
				req.CustomerID = &actor.Customer.ID
			} else if actor.AnonymousID != "" {
				req.AnonymousID = &actor.AnonymousID
			}
			cart, err := deps.CartSvc.Create(c.Request.Context(), project.ID, req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, toCTCart(*cart, actor.Customer))
		})
		group.POST("/me/carts/:id", func(c *gin.Context) {
			project := mustProject(c)
			actor, ok := authorizeActor(c, project, deps.CustomerSvc, deps.AnonymousSvc)
			if !ok {
				return
			}
			id := c.Param("id")
			var req cartsvc.UpdateInput
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
				return
			}
			var cart *domain.Cart
			var err error
			if actor.Customer != nil {
				cart, err = deps.CartSvc.Update(c.Request.Context(), project.ID, actor.Customer.ID, id, req)
			} else {
				cart, err = deps.CartSvc.UpdateAnonymous(c.Request.Context(), project.ID, actor.AnonymousID, id, req)
			}
			if err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					c.JSON(http.StatusNotFound, gin.H{"error": "cart not found"})
					return
				}
				logger.Printf("cart update error project_id=%s cart_id=%s error=%v", project.ID, id, err)
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, toCTCart(*cart, actor.Customer))
		})
		group.DELETE("/me/carts/:id", func(c *gin.Context) {
			project := mustProject(c)
			actor, ok := authorizeActor(c, project, deps.CustomerSvc, deps.AnonymousSvc)
			if !ok {
				return
			}
			id := c.Param("id")
			var cart *domain.Cart
			var err error
			if actor.Customer != nil {
				cart, err = deps.CartSvc.Delete(c.Request.Context(), project.ID, actor.Customer.ID, id)
			} else {
				cart, err = deps.CartSvc.DeleteAnonymous(c.Request.Context(), project.ID, actor.AnonymousID, id)
			}
			if err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					c.JSON(http.StatusNotFound, gin.H{"error": "cart not found"})
					return
				}
				logger.Printf("cart delete error project_id=%s cart_id=%s error=%v", project.ID, id, err)
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, toCTCart(*cart, actor.Customer))
		})
		group.GET("/me/active-cart", func(c *gin.Context) {
			project := mustProject(c)
			actor, ok := authorizeActor(c, project, deps.CustomerSvc, deps.AnonymousSvc)
			if !ok {
				return
			}
			var cart *domain.Cart
			var err error
			if actor.Customer != nil {
				cart, err = deps.CartSvc.GetActive(c.Request.Context(), project.ID, actor.Customer.ID)
			} else {
				cart, err = deps.CartSvc.GetActiveAnonymous(c.Request.Context(), project.ID, actor.AnonymousID)
			}
			if err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					c.JSON(http.StatusNotFound, gin.H{"error": "cart not found"})
					return
				}
				logger.Printf("active cart error project_id=%s error=%v", project.ID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "get active cart failed"})
				return
			}
			c.JSON(http.StatusOK, toCTCart(*cart, actor.Customer))
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

	oauth := router.Group("/oauth/:projectKey", projectMiddleware(logger, deps.ProjectRepo))
	oauth.POST("/customers/token", func(c *gin.Context) {
		project := mustProject(c)

		var req tokenRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token request"})
			return
		}
		if strings.ToLower(req.GrantType) != "password" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported grant_type"})
			return
		}
		if !strings.Contains(req.Scope, "manage_project:"+project.Key) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scope"})
			return
		}

		customer, accessToken, refreshToken, err := deps.CustomerSvc.Login(c.Request.Context(), project.ID, req.Username, req.Password)
		if err != nil {
			status := http.StatusUnauthorized
			msg := "invalid credentials"
			if err != customersvc.ErrInvalidCredentials {
				status = http.StatusInternalServerError
				msg = "token issuance failed"
			}
			c.JSON(status, gin.H{"error": msg})
			return
		}

		scope := "manage_project:" + project.Key + " customer_id:" + customer.ID
		c.JSON(http.StatusOK, gin.H{
			"access_token":  accessToken,
			"expires_in":    deps.CustomerSvc.AccessTTLSeconds(),
			"token_type":    "Bearer",
			"scope":         scope,
			"refresh_token": refreshToken,
		})
	})
	oauth.POST("/anonymous/token", func(c *gin.Context) {
		project := mustProject(c)

		var req anonymousTokenRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token request"})
			return
		}
		if strings.ToLower(req.GrantType) != "client_credentials" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported grant_type"})
			return
		}
		if !strings.Contains(req.Scope, "manage_project:"+project.Key) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scope"})
			return
		}

		accessToken, refreshToken, anonymousID, err := deps.AnonymousSvc.Issue(c.Request.Context(), project.ID)
		if err != nil {
			logger.Printf("anonymous token error project_id=%s error=%v", project.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token issuance failed"})
			return
		}

		scope := "manage_project:" + project.Key + " anonymous_id:" + anonymousID
		c.JSON(http.StatusOK, gin.H{
			"access_token":  accessToken,
			"expires_in":    deps.AnonymousSvc.AccessTTLSeconds(),
			"token_type":    "Bearer",
			"scope":         scope,
			"refresh_token": refreshToken,
		})
	})

	router.POST("/oauth/token", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"access_token": "NOT_IMPLEMENTED",
			"expires_in":   172800,
			"token_type":   "Bearer",
			"scope":        "manage_project:petal_pot",
		})
	})

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

func authorizeCustomer(c *gin.Context, project *domain.Project, svc customerService) (*domain.Customer, bool) {
	authHeader := c.GetHeader("Authorization")
	token := extractBearerToken(authHeader)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return nil, false
	}
	customer, err := svc.LookupByToken(c.Request.Context(), project.ID, token)
	if err != nil {
		status := http.StatusUnauthorized
		msg := "invalid token"
		if !errors.Is(err, customersvc.ErrInvalidToken) {
			status = http.StatusInternalServerError
			msg = "failed to resolve customer"
		}
		c.JSON(status, gin.H{"error": msg})
		return nil, false
	}
	return customer, true
}

type authActor struct {
	Customer    *domain.Customer
	AnonymousID string
}

func authorizeActor(c *gin.Context, project *domain.Project, custSvc customerService, anonSvc anonymousService) (*authActor, bool) {
	authHeader := c.GetHeader("Authorization")
	token := extractBearerToken(authHeader)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return nil, false
	}
	customer, err := custSvc.LookupByToken(c.Request.Context(), project.ID, token)
	if err == nil {
		return &authActor{Customer: customer}, true
	}
	if !errors.Is(err, customersvc.ErrInvalidToken) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve customer"})
		return nil, false
	}

	anonymousID, err := anonSvc.LookupByToken(c.Request.Context(), project.ID, token)
	if err == nil {
		return &authActor{AnonymousID: anonymousID}, true
	}
	if !errors.Is(err, anonymoussvc.ErrInvalidToken) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve anonymous token"})
		return nil, false
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
	return nil, false
}

func extractBearerToken(authHeader string) string {
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return ""
	}
	return strings.TrimSpace(authHeader[len("Bearer "):])
}
