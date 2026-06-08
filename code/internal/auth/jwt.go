package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

func InitJWT(secret string) {
	jwtSecret = []byte(secret)
}

func GenerateJWT(tenantID string) (string, error) {
	if len(jwtSecret) == 0 {
		return "", errors.New("jwt secret not initialized")
	}
	claims := jwt.MapClaims{
		"sub": tenantID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ValidateJWT(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token")
	}
	sub, _ := claims.GetSubject()
	if sub == "" {
		return "", errors.New("missing sub claim")
	}
	return sub, nil
}

type AuthClaims struct {
	UserID   string
	TenantID string
	Role     string
}

func GenerateUserJWT(userID string) (string, error) {
	if len(jwtSecret) == 0 {
		return "", errors.New("jwt secret not initialized")
	}
	claims := jwt.MapClaims{
		"sub":  userID,
		"type": "user",
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func GenerateScopedJWT(userID, tenantID, role string) (string, error) {
	if len(jwtSecret) == 0 {
		return "", errors.New("jwt secret not initialized")
	}
	claims := jwt.MapClaims{
		"sub":  userID,
		"tid":  tenantID,
		"role": role,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ValidateJWTClaims(tokenStr string) (*AuthClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	sub, _ := claims.GetSubject()
	if sub == "" {
		return nil, errors.New("missing sub claim")
	}

	ac := &AuthClaims{}
	if tid, ok := claims["tid"].(string); ok && tid != "" {
		ac.UserID = sub
		ac.TenantID = tid
		if role, ok := claims["role"].(string); ok {
			ac.Role = role
		}
	} else if t, ok := claims["type"].(string); ok && t == "user" {
		ac.UserID = sub
	} else {
		ac.TenantID = sub
	}
	return ac, nil
}
