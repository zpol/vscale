package utils

import (
    "errors"
    "os"
    "time"

    "github.com/golang-jwt/jwt/v4"
)

type Claims struct {
    UserID uint `json:"user_id"`
    jwt.StandardClaims
}

func getEnv(key, defaultValue string) string {
    value, exists := os.LookupEnv(key)
    if !exists {
        return defaultValue
    }
    return value
}

var jwtKey = []byte(getEnv("JWT_SECRET", "your_very_long_and_secure_jwt_secret_here"))

func ParseToken(tokenString string) (*Claims, error) {
    claims := &Claims{}
    token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
        return jwtKey, nil
    })
    if err != nil || !token.Valid {
        return nil, errors.New("invalid token")
    }
    return claims, nil
}
