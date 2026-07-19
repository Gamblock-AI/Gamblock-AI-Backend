package model

import "github.com/golang-jwt/jwt/v5"

type Claims struct {
	UserID   string           `json:"uid"`
	Email    string           `json:"email"`
	Role     string           `json:"role"`
	AuthTime *jwt.NumericDate `json:"auth_time"`
	jwt.RegisteredClaims
}
