package jwt

import (
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/moonwalker/moonbase/internal/jwe"
)

type AuthClaims struct {
	jwt.StandardClaims
	Data []byte `json:"data"`
}

func EncryptAndSign(encKey, sigKey []byte, data []byte, expires time.Duration) (string, error) {
	encData, err := jwe.Encrypt(encKey, data)
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, &AuthClaims{
		Data: encData,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  time.Now().Unix(),
			NotBefore: time.Now().Unix(),
			ExpiresAt: time.Now().Add(expires).Unix(),
		},
	})

	return token.SignedString(sigKey)
}

func VerifyAndDecrypt(encKey, sigKey []byte, tokenString string) (*jwt.Token, error) {
	claims := &AuthClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return sigKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims.Data, err = jwe.Decrypt(encKey, claims.Data)
	if err != nil {
		return nil, err
	}

	return token, nil
}
