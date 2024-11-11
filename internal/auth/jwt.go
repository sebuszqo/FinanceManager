package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt"
	"log"
	"os"
	"time"
)

var (
	ErrInvalidJWTToken        = errors.New("JWT token is invalid")
	ErrExpiredJWTToken        = errors.New("JWT token is expired")
	ErrInvalidJWTRefreshToken = errors.New("JWT Refresh token is invalid")
)

const defaultJWTRefreshDuration = 720 * time.Hour
const defaultJWTDuration = 10 * time.Minute

type JWTManagerInterface interface {
	GenerateAccessJWT(user string, duration time.Duration) (string, error)
	ValidateAccessToken(tokenString string) (string, error)
	GenerateRefreshJWT(userID, tokenHash string, duration time.Duration) (string, error)
	ValidateRefreshToken(tokenString, tokenHash string) error
	ExtractUserIDFromRefreshToken(tokenString string) (string, error)
}

type AccessTokenCustomClaims struct {
	UserID string `json:"user_id"`
	jwt.StandardClaims
}

type RefreshTokenCustomClaims struct {
	UserID string `json:"user_id"`
	CusKey string `json:"cus_key"`
	jwt.StandardClaims
}

type JWTManager struct {
	secret string
}

func NewJWTManager() JWTManagerInterface {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatalf("JWT_SECRET is not set in .env file")
	}

	return &JWTManager{
		secret: jwtSecret,
	}
}

func (j *JWTManager) generateCustomKey(userID string, tokenHash string) string {

	// data := userID + tokenHash
	h := hmac.New(sha256.New, []byte(tokenHash))
	h.Write([]byte(userID))
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}

func (j *JWTManager) GenerateRefreshJWT(userID, tokenHash string, duration time.Duration) (string, error) {
	cusKey := j.generateCustomKey(userID, tokenHash)
	claims := &RefreshTokenCustomClaims{
		UserID: userID,
		CusKey: cusKey,
		StandardClaims: jwt.StandardClaims{
			Subject:   userID,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(duration).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secret))
}

func (j *JWTManager) GenerateAccessJWT(userID string, duration time.Duration) (string, error) {
	claims := &AccessTokenCustomClaims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			Subject:   userID,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(duration).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secret))
}

func (j *JWTManager) ValidateAccessToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessTokenCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.secret), nil
	})

	if err != nil {
		var validationErr *jwt.ValidationError
		if errors.As(err, &validationErr) {
			if validationErr.Errors&(jwt.ValidationErrorExpired) != 0 {
				return "", ErrExpiredJWTToken
			}
		}
		return "", err
	}

	claims, ok := token.Claims.(*AccessTokenCustomClaims)
	if !ok || !token.Valid || claims.UserID == "" {
		return "", ErrInvalidJWTToken
	}

	return claims.UserID, nil
}

func (j *JWTManager) ExtractUserIDFromRefreshToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.secret), nil
	})

	if err != nil {
		var validationErr *jwt.ValidationError
		if errors.As(err, &validationErr) {
			if validationErr.Errors&(jwt.ValidationErrorExpired) != 0 {
				return "", ErrExpiredJWTToken
			}
		}
		return "", err
	}

	claims, ok := token.Claims.(*RefreshTokenCustomClaims)
	if !ok || !token.Valid || claims.UserID == "" {
		return "", ErrInvalidJWTToken
	}

	return claims.UserID, nil
}

func (j *JWTManager) ValidateRefreshToken(tokenString, tokenHash string) error {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.secret), nil
	})

	if err != nil {
		var validationErr *jwt.ValidationError
		if errors.As(err, &validationErr) {
			if validationErr.Errors&(jwt.ValidationErrorExpired) != 0 {
				return ErrExpiredJWTToken
			}
		}
		return err
	}

	claims, ok := token.Claims.(*RefreshTokenCustomClaims)
	if !ok || !token.Valid || claims.UserID == "" {
		return ErrInvalidJWTToken
	}

	// should I validate cus here or not that's the question
	expectedCusKey := j.generateCustomKey(claims.UserID, tokenHash)
	if claims.CusKey != expectedCusKey {
		fmt.Println("custom key in refresh token is not valid!")
		return ErrInvalidJWTToken
	}

	return nil
}
