package auth

import (
	"errors"
	"github.com/golang-jwt/jwt"
	"time"
)

var (
	ErrInvalidJWTToken = errors.New("JWT token is invalid")
	ErrExpiredJWTToken = errors.New("JWT token is expired")
)

const defaultJWTDuration = 1 * time.Hour

type JWTManagerInterface interface {
	GenerateJWT(user *User, duration time.Duration) (string, error)
	VerifyJWT(tokenString string) (string, error)
}

type JWTManager struct{}

func NewJWTManager() JWTManagerInterface {
	return &JWTManager{}
}

func (j *JWTManager) GenerateJWT(user *User, duration time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(duration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func (j *JWTManager) VerifyJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
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

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["user_id"].(string)
		if !ok {
			return "", ErrInvalidJWTToken
		}
		return userID, nil
	}

	return "", ErrInvalidJWTToken
}
