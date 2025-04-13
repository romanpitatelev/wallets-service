//nolint:testpackage
package tests

import (
	"crypto/rsa"
	_ "embed"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/romanpitatelev/wallets-service/internal/entity"
)

//go:embed keys/private_key.pem
var privateKeyData []byte

func readPrivateKey() (*rsa.PrivateKey, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
	if err != nil {
		return nil, fmt.Errorf("error parsing private key: %w", err)
	}

	return privateKey, nil
}

func generateToken(claims *entity.Claims, secret *rsa.PrivateKey) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	tokenStr, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenStr, nil
}

func ValidateToken(tokenStr string, claims *entity.Claims, secret *rsa.PublicKey) error {
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if method, ok := token.Method.(*jwt.SigningMethodRSA); !ok || method != jwt.SigningMethodRS256 {
			return nil, entity.ErrInvalidSigningMethod
		}

		return secret, nil
	})
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return entity.ErrInvalidToken
	}

	return nil
}
