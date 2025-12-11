package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"commercetools-replica/internal/domain"
	"github.com/gin-gonic/gin"
)

type stubProjectRepo struct {
	project *domain.Project
	err     error
}

func (s *stubProjectRepo) GetByKey(_ context.Context, _ string) (*domain.Project, error) {
	return s.project, s.err
}

func TestProjectMiddleware_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubProjectRepo{
		project: &domain.Project{ID: "123", Key: "proj", Name: "Test"},
	}
	router := gin.New()
	router.Use(projectMiddleware(repo))
	router.GET("/projects/:projectKey/test", func(c *gin.Context) {
		p := c.Request.Context().Value(projectCtxKey)
		if p == nil {
			t.Fatalf("expected project in context")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/proj/test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestProjectMiddleware_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubProjectRepo{err: domain.ErrNotFound}
	router := gin.New()
	router.Use(projectMiddleware(repo))
	router.GET("/projects/:projectKey/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/missing/test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestProjectMiddleware_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubProjectRepo{err: errors.New("boom")}
	router := gin.New()
	router.Use(projectMiddleware(repo))
	router.GET("/projects/:projectKey/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/proj/test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestProjectMiddleware_MissingKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubProjectRepo{}
	router := gin.New()
	router.Use(projectMiddleware(repo))
	router.GET("/projects/:projectKey/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/projects//test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}
