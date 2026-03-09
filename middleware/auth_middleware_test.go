package middleware

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddlewareMissingHeader(t *testing.T) {
	privateKey := mustGenerateKey(t)
	pub := privateKey.PublicKey
	jwks := buildJWKS("kid-1", &pub)
	baseURL := "https://example.supabase.co"
	restoreEnv := setEnv(t, "SUPABASE_URL", baseURL)
	defer restoreEnv()
	restoreHTTP := setHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.String() == baseURL+"/auth/v1/.well-known/jwks.json" {
			return newJSONResponse(http.StatusOK, jwks), nil
		}
		return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
	})
	defer restoreHTTP()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/secure", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddlewareValidToken(t *testing.T) {
	privateKey := mustGenerateKey(t)
	pub := privateKey.PublicKey
	jwks := buildJWKS("kid-1", &pub)
	baseURL := "https://example.supabase.co"
	restoreEnv := setEnv(t, "SUPABASE_URL", baseURL)
	defer restoreEnv()
	restoreHTTP := setHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.String() == baseURL+"/auth/v1/.well-known/jwks.json" {
			return newJSONResponse(http.StatusOK, jwks), nil
		}
		return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
	})
	defer restoreHTTP()

	tokenString := mustSignToken(t, privateKey, "kid-1", baseURL+"/auth/v1", "user-123")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/secure", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"userID": c.GetString("userID")})
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "user-123") {
		t.Fatalf("expected userID in response, got %s", w.Body.String())
	}
}

func TestAuthMiddlewareInvalidIssuer(t *testing.T) {
	privateKey := mustGenerateKey(t)
	pub := privateKey.PublicKey
	jwks := buildJWKS("kid-1", &pub)
	baseURL := "https://example.supabase.co"
	restoreEnv := setEnv(t, "SUPABASE_URL", baseURL)
	defer restoreEnv()
	restoreHTTP := setHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.String() == baseURL+"/auth/v1/.well-known/jwks.json" {
			return newJSONResponse(http.StatusOK, jwks), nil
		}
		return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
	})
	defer restoreHTTP()

	tokenString := mustSignToken(t, privateKey, "kid-1", "https://wrong.example.com/auth/v1", "user-123")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/secure", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestJWKToECDSAPublicKeyInvalidPoint(t *testing.T) {
	key := jwkKey{
		Kid: "kid-1",
		Kty: "EC",
		Crv: "P-256",
		X:   base64.RawURLEncoding.EncodeToString([]byte{1}),
		Y:   base64.RawURLEncoding.EncodeToString([]byte{1}),
	}

	_, err := jwkToECDSAPublicKey(key)
	if err == nil {
		t.Fatal("expected error for invalid curve point")
	}
}

func mustGenerateKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	return key
}

func buildJWKS(kid string, pub *ecdsa.PublicKey) string {
	x := base64.RawURLEncoding.EncodeToString(pub.X.Bytes())
	y := base64.RawURLEncoding.EncodeToString(pub.Y.Bytes())
	return fmt.Sprintf(`{"keys":[{"kid":"%s","kty":"EC","crv":"P-256","x":"%s","y":"%s"}]}`, kid, x, y)
}

func mustSignToken(t *testing.T, privateKey *ecdsa.PrivateKey, kid, issuer, sub string) string {
	t.Helper()
	claims := SupabaseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   sub,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}

func setEnv(t *testing.T, key, value string) func() {
	t.Helper()
	old, had := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	return func() {
		var err error
		if had {
			err = os.Setenv(key, old)
		} else {
			err = os.Unsetenv(key)
		}
		if err != nil {
			t.Fatalf("failed to restore env: %v", err)
		}
	}
}

func TestFetchJWKSErrorsOnBadStatus(t *testing.T) {
	jwksURL := "https://example.supabase.co/auth/v1/.well-known/jwks.json"
	restoreHTTP := setHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.String() == jwksURL {
			return newJSONResponse(http.StatusInternalServerError, `{"error":"boom"}`), nil
		}
		return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
	})
	defer restoreHTTP()

	_, err := fetchJWKS(jwksURL)
	if err == nil {
		t.Fatal("expected error on non-200 JWKS response")
	}
}

func TestFetchJWKSBuildsKeyMap(t *testing.T) {
	priv := mustGenerateKey(t)
	x := base64.RawURLEncoding.EncodeToString(priv.PublicKey.X.Bytes())
	y := base64.RawURLEncoding.EncodeToString(priv.PublicKey.Y.Bytes())
	jwksURL := "https://example.supabase.co/auth/v1/.well-known/jwks.json"
	restoreHTTP := setHTTPTransport(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.String() == jwksURL {
			body := fmt.Sprintf(`{"keys":[{"kid":"k1","kty":"EC","crv":"P-256","x":"%s","y":"%s"}]}`, x, y)
			return newJSONResponse(http.StatusOK, body), nil
		}
		return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
	})
	defer restoreHTTP()

	keys, err := fetchJWKS(jwksURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if _, ok := keys["k1"]; !ok {
		t.Fatalf("expected key k1 in result map")
	}
}

func TestJWKToECDSAPublicKeyValid(t *testing.T) {
	priv := mustGenerateKey(t)
	key := jwkKey{
		Kid: "k1",
		Kty: "EC",
		Crv: "P-256",
		X:   base64.RawURLEncoding.EncodeToString(priv.PublicKey.X.Bytes()),
		Y:   base64.RawURLEncoding.EncodeToString(priv.PublicKey.Y.Bytes()),
	}

	got, err := jwkToECDSAPublicKey(key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.X.Cmp(new(big.Int).Set(priv.PublicKey.X)) != 0 {
		t.Fatalf("unexpected X coordinate")
	}
	if got.Y.Cmp(new(big.Int).Set(priv.PublicKey.Y)) != 0 {
		t.Fatalf("unexpected Y coordinate")
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func setHTTPTransport(t *testing.T, fn roundTripFunc) func() {
	t.Helper()
	old := http.DefaultTransport
	http.DefaultTransport = fn
	return func() {
		http.DefaultTransport = old
	}
}

func newJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
