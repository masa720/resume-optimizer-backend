package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/masa720/resume-optimizer-backend/domain"
	"github.com/masa720/resume-optimizer-backend/service"
)

type mockAnalysisRepo struct {
	createFn        func(analysis *domain.Analysis) error
	getByIDFn       func(userID, analysisID string) (*domain.Analysis, error)
	getAllByUserFn  func(userID string, query domain.ListQuery) (*domain.ListResult, error)
	deleteFn        func(userID, analysisID string) error
	updateStatusFn  func(userID, analysisID, status string) (*domain.Analysis, error)
	createVersionFn func(version *domain.AnalysisVersion) error
	getMaxVersionFn func(analysisID string) (int, error)
}

type mockSuggestionProvider struct {
	generateFn func(ctx context.Context, jobDescription string, resumeText string, missingKeywords pq.StringArray) ([]string, error)
}

func (m *mockSuggestionProvider) Generate(ctx context.Context, jobDescription string, resumeText string, missingKeywords pq.StringArray) ([]string, error) {
	if m.generateFn != nil {
		return m.generateFn(ctx, jobDescription, resumeText, missingKeywords)
	}
	return []string{"fallback suggestion"}, nil
}

func (m *mockAnalysisRepo) Create(analysis *domain.Analysis) error {
	if m.createFn != nil {
		return m.createFn(analysis)
	}
	return nil
}

func (m *mockAnalysisRepo) GetByID(userID, analysisID string) (*domain.Analysis, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(userID, analysisID)
	}
	return nil, nil
}

func (m *mockAnalysisRepo) GetAllByUserID(userID string, query domain.ListQuery) (*domain.ListResult, error) {
	if m.getAllByUserFn != nil {
		return m.getAllByUserFn(userID, query)
	}
	return &domain.ListResult{}, nil
}

func (m *mockAnalysisRepo) Delete(userID, analysisID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(userID, analysisID)
	}
	return nil
}

func (m *mockAnalysisRepo) UpdateStatus(userID, analysisID, status string) (*domain.Analysis, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(userID, analysisID, status)
	}
	return &domain.Analysis{Status: status}, nil
}

func (m *mockAnalysisRepo) CreateVersion(version *domain.AnalysisVersion) error {
	if m.createVersionFn != nil {
		return m.createVersionFn(version)
	}
	return nil
}

func (m *mockAnalysisRepo) GetMaxVersion(analysisID string) (int, error) {
	if m.getMaxVersionFn != nil {
		return m.getMaxVersionFn(analysisID)
	}
	return 0, nil
}

func setupAnalysisRouter(repo domain.AnalysisRepository) *gin.Engine {
	return setupAnalysisRouterWithSuggestionProvider(repo, &mockSuggestionProvider{})
}

func setupAnalysisRouterWithSuggestionProvider(repo domain.AnalysisRepository, provider service.SuggestionProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAnalysisHandler(repo, provider, &service.FallbackJDAnalyzer{}, nil)
	auth := func(c *gin.Context) {
		c.Set("userID", "user-1")
		c.Next()
	}
	r.POST("/analyses", auth, h.Create)
	r.GET("/analyses", auth, h.List)
	r.GET("/analyses/:id", auth, h.GetByID)
	r.DELETE("/analyses/:id", auth, h.Delete)
	return r
}

func TestAnalysisCreateSuccess(t *testing.T) {
	repo := &mockAnalysisRepo{
		createFn: func(analysis *domain.Analysis) error {
			analysis.ID = "analysis-1"
			return nil
		},
	}
	r := setupAnalysisRouter(repo)

	body := []byte(`{"jobDescription":"go backend","resumeText":"go rest api","companyName":"A","jobPosition":"B"}`)
	req := httptest.NewRequest(http.MethodPost, "/analyses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}
}

func TestAnalysisCreateBadRequest(t *testing.T) {
	repo := &mockAnalysisRepo{}
	r := setupAnalysisRouter(repo)

	body := []byte(`{"jobDescription":"missing resumeText"}`)
	req := httptest.NewRequest(http.MethodPost, "/analyses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestAnalysisCreateInternalError(t *testing.T) {
	repo := &mockAnalysisRepo{
		createFn: func(analysis *domain.Analysis) error {
			return errors.New("db error")
		},
	}
	r := setupAnalysisRouter(repo)

	body := []byte(`{"jobDescription":"go backend","resumeText":"go rest api"}`)
	req := httptest.NewRequest(http.MethodPost, "/analyses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

func TestAnalysisListSuccess(t *testing.T) {
	repo := &mockAnalysisRepo{
		getAllByUserFn: func(userID string, query domain.ListQuery) (*domain.ListResult, error) {
			return &domain.ListResult{
				Data:       []domain.Analysis{{ID: "a1"}, {ID: "a2"}},
				TotalCount: 2,
				Page:       query.Page,
				Limit:      query.Limit,
				TotalPages: 1,
			}, nil
		},
	}
	r := setupAnalysisRouter(repo)
	req := httptest.NewRequest(http.MethodGet, "/analyses", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestAnalysisGetByIDNotFound(t *testing.T) {
	repo := &mockAnalysisRepo{
		getByIDFn: func(userID, analysisID string) (*domain.Analysis, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	r := setupAnalysisRouter(repo)
	req := httptest.NewRequest(http.MethodGet, "/analyses/a1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestAnalysisGetByIDInternalError(t *testing.T) {
	repo := &mockAnalysisRepo{
		getByIDFn: func(userID, analysisID string) (*domain.Analysis, error) {
			return nil, errors.New("db down")
		},
	}
	r := setupAnalysisRouter(repo)
	req := httptest.NewRequest(http.MethodGet, "/analyses/a1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

func TestAnalysisDeleteNoContent(t *testing.T) {
	repo := &mockAnalysisRepo{
		deleteFn: func(userID, analysisID string) error { return nil },
	}
	r := setupAnalysisRouter(repo)
	req := httptest.NewRequest(http.MethodDelete, "/analyses/a1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", w.Code)
	}
}

func TestAnalysisDeleteNotFound(t *testing.T) {
	repo := &mockAnalysisRepo{
		deleteFn: func(userID, analysisID string) error { return gorm.ErrRecordNotFound },
	}
	r := setupAnalysisRouter(repo)
	req := httptest.NewRequest(http.MethodDelete, "/analyses/a1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestAnalysisDeleteInternalError(t *testing.T) {
	repo := &mockAnalysisRepo{
		deleteFn: func(userID, analysisID string) error { return errors.New("db down") },
	}
	r := setupAnalysisRouter(repo)
	req := httptest.NewRequest(http.MethodDelete, "/analyses/a1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

func TestAnalysisUnauthorizedWhenUserMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAnalysisHandler(&mockAnalysisRepo{}, &mockSuggestionProvider{}, &service.FallbackJDAnalyzer{}, nil)
	r.GET("/analyses", h.List)

	req := httptest.NewRequest(http.MethodGet, "/analyses", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}

	var got map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if got["error"] != "unauthorized" {
		t.Fatalf("expected unauthorized error, got %q", got["error"])
	}
}

func TestAnalysisCreateFallsBackWhenProviderFails(t *testing.T) {
	repo := &mockAnalysisRepo{
		createFn: func(analysis *domain.Analysis) error {
			return nil
		},
	}
	provider := &mockSuggestionProvider{
		generateFn: func(ctx context.Context, jobDescription string, resumeText string, missingKeywords pq.StringArray) ([]string, error) {
			return nil, errors.New("provider down")
		},
	}
	r := setupAnalysisRouterWithSuggestionProvider(repo, provider)

	body := []byte(`{"jobDescription":"go backend docker","resumeText":"go backend"}`)
	req := httptest.NewRequest(http.MethodPost, "/analyses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}
}

func TestAnalysisCreateStoresResultInVersion(t *testing.T) {
	var capturedVersion *domain.AnalysisVersion
	repo := &mockAnalysisRepo{
		createFn: func(analysis *domain.Analysis) error {
			analysis.ID = "analysis-1"
			return nil
		},
		createVersionFn: func(version *domain.AnalysisVersion) error {
			capturedVersion = version
			return nil
		},
	}
	r := setupAnalysisRouter(repo)

	body := []byte(`{"jobDescription":"backend docker","resumeText":"backend"}`)
	req := httptest.NewRequest(http.MethodPost, "/analyses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}
	if capturedVersion == nil {
		t.Fatal("expected version to be created")
	}
	if capturedVersion.Version != 1 {
		t.Fatalf("expected version 1, got %d", capturedVersion.Version)
	}
	if capturedVersion.MatchScore != 50 {
		t.Fatalf("expected match score 50, got %d", capturedVersion.MatchScore)
	}
}
