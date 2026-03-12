package handler

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/masa720/resume-optimizer-backend/domain"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type mockProfileRepo struct {
	getByIDFn func(userID string) (*domain.Profile, error)
	updateFn  func(profile *domain.Profile) error
}

func (m *mockProfileRepo) GetByID(userID string) (*domain.Profile, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(userID)
	}
	return nil, nil
}

func (m *mockProfileRepo) Update(profile *domain.Profile) error {
	if m.updateFn != nil {
		return m.updateFn(profile)
	}
	return nil
}

func setupProfileRouter(repo domain.ProfileRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewProfileHandler(repo)
	auth := func(c *gin.Context) {
		c.Set("userID", "user-1")
		c.Next()
	}
	r.GET("/profile", auth, h.GetProfile)
	r.PUT("/profile", auth, h.UpdateProfile)
	return r
}

func TestProfileGetSuccess(t *testing.T) {
	repo := &mockProfileRepo{
		getByIDFn: func(userID string) (*domain.Profile, error) {
			return &domain.Profile{ID: userID, Username: "testUser"}, nil
		},
	}
	r := setupProfileRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestProfileGetNotFound(t *testing.T) {
	repo := &mockProfileRepo{
		getByIDFn: func(userID string) (*domain.Profile, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	r := setupProfileRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestProfileGetInternalError(t *testing.T) {
	repo := &mockProfileRepo{
		getByIDFn: func(userID string) (*domain.Profile, error) {
			return nil, errors.New("db down")
		},
	}
	r := setupProfileRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

func TestProfileUpdateBadRequest(t *testing.T) {
	repo := &mockProfileRepo{}
	r := setupProfileRouter(repo)

	req := httptest.NewRequest(http.MethodPut, "/profile", bytes.NewBufferString(`{"invalid":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestProfileUpdateUsernameRequired(t *testing.T) {
	repo := &mockProfileRepo{}
	r := setupProfileRouter(repo)

	req := httptest.NewRequest(http.MethodPut, "/profile", bytes.NewBufferString(`{"username":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestProfileUpdateCreatesWhenMissing(t *testing.T) {
	repo := &mockProfileRepo{
		getByIDFn: func(userID string) (*domain.Profile, error) {
			return nil, gorm.ErrRecordNotFound
		},
		updateFn: func(profile *domain.Profile) error {
			if profile.ID != "user-1" || profile.Username != "testUser" {
				t.Fatalf("unexpected profile payload: %+v", profile)
			}
			return nil
		},
	}
	r := setupProfileRouter(repo)

	req := httptest.NewRequest(http.MethodPut, "/profile", bytes.NewBufferString(`{"username":"testUser"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}
}

func TestProfileUpdateExisting(t *testing.T) {
	repo := &mockProfileRepo{
		getByIDFn: func(userID string) (*domain.Profile, error) {
			return &domain.Profile{ID: userID, Username: "old"}, nil
		},
		updateFn: func(profile *domain.Profile) error {
			if profile.Username != "new-name" {
				t.Fatalf("expected updated username, got %q", profile.Username)
			}
			return nil
		},
	}
	r := setupProfileRouter(repo)

	req := httptest.NewRequest(http.MethodPut, "/profile", bytes.NewBufferString(`{"username":"new-name"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestProfileUpdateInternalErrorOnCreate(t *testing.T) {
	repo := &mockProfileRepo{
		getByIDFn: func(userID string) (*domain.Profile, error) {
			return nil, gorm.ErrRecordNotFound
		},
		updateFn: func(profile *domain.Profile) error {
			return errors.New("write failed")
		},
	}
	r := setupProfileRouter(repo)

	req := httptest.NewRequest(http.MethodPut, "/profile", bytes.NewBufferString(`{"username":"testUser"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

func TestProfileUnauthorizedWhenUserMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewProfileHandler(&mockProfileRepo{})
	r.GET("/profile", h.GetProfile)

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}
