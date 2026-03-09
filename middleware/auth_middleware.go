package middleware

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type SupabaseClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email,omitempty"`
	Role  string `json:"role,omitempty"`
}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func AuthMiddleware() gin.HandlerFunc {
	supabaseURL := strings.TrimRight(os.Getenv("SUPABASE_URL"), "/")
	if supabaseURL == "" {
		panic("SUPABASE_URL is required")
	}

	jwksURL := supabaseURL + "/auth/v1/.well-known/jwks.json"
	keys, err := fetchJWKS(jwksURL)
	if err != nil {
		panic("failed to initialize JWKS: " + err.Error())
	}

	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			ctx.Abort()
			return
		}

		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			ctx.Abort()
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
		if tokenString == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "empty token"})
			ctx.Abort()
			return
		}

		claims := &SupabaseClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodES256.Alg() {
				return nil, fmt.Errorf("unexpected signing algorithm: %s", token.Method.Alg())
			}

			kid, _ := token.Header["kid"].(string)
			if kid == "" {
				return nil, errors.New("missing kid")
			}

			key, ok := keys[kid]
			if !ok {
				return nil, errors.New("key not found for kid")
			}

			return key, nil
		})
		if err != nil || !token.Valid {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			ctx.Abort()
			return
		}

		issuer := supabaseURL + "/auth/v1"
		if claims.Issuer != issuer || claims.Subject == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			ctx.Abort()
			return
		}

		ctx.Set("userID", claims.Subject)
		ctx.Next()
	}
}

func fetchJWKS(jwksURL string) (map[string]*ecdsa.PublicKey, error) {
	resp, err := http.Get(jwksURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected JWKS status: %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}

	keys := make(map[string]*ecdsa.PublicKey)
	for _, key := range jwks.Keys {
		if key.Kty != "EC" || key.Crv != "P-256" || key.Kid == "" {
			continue
		}

		pubKey, err := jwkToECDSAPublicKey(key)
		if err != nil {
			continue
		}

		keys[key.Kid] = pubKey
	}

	if len(keys) == 0 {
		return nil, errors.New("no usable keys in JWKS")
	}

	return keys, nil
}

func jwkToECDSAPublicKey(key jwkKey) (*ecdsa.PublicKey, error) {
	xBytes, err := base64.RawURLEncoding.DecodeString(key.X)
	if err != nil {
		return nil, err
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(key.Y)
	if err != nil {
		return nil, err
	}

	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)

	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}

	if !pubKey.Curve.IsOnCurve(pubKey.X, pubKey.Y) {
		return nil, errors.New("jwk key is not on P-256 curve")
	}

	return pubKey, nil
}
