package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// GetJWTToken generates a JWT token for the specified service using the private key at the given path.
func GetJWTToken(serviceName, keyPath string) (string, error) {
	privKey, err := LoadPrivateKey(keyPath)
	if err != nil {
		return "", fmt.Errorf("failed to load private key: %v", err)
	}

	jwtToken, err := createJWTToken(serviceName, privKey)
	if err != nil {
		return "", err
	}

	return jwtToken, nil
}

// LoadPrivateKey loads the ECDSA private key from the specified path.
func LoadPrivateKey(keyPath string) (*ecdsa.PrivateKey, error) {
	file, err := os.Open(keyPath)
	if !errors.Is(err, os.ErrNotExist) && err != nil {
		return nil, err
	}

	defer file.Close()

	if errors.Is(err, os.ErrNotExist) {
		// Generate an ECDSA private key
		privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key: %v", err)
		}

		// Serialize the private key to PEM
		privBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal private key: %v", err)
		}
		privPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: privBytes,
		})

		serviceSecret := map[string]string{
			"private_key": string(privPEM),
		}

		file, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		if err := encoder.Encode(&serviceSecret); err != nil {
			return nil, err
		}

		return privKey, nil
	}

	return loadPrivateKey(file)
}

// ExportPublicKey exports the public key from the private key stored at the specified path.
func ExportPublicKey(keyPath string) (string, error) {
	file, err := os.Open(keyPath)
	if err != nil {
		return "", err
	}

	defer file.Close()

	privKey, err := loadPrivateKey(file)
	if err != nil {
		return "", fmt.Errorf("failed to load private key: %v", err)
	}

	// Serialize the public key to PEM
	pubBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	return string(pubPEM), nil
}

func loadPrivateKey(r io.Reader) (*ecdsa.PrivateKey, error) {
	var serviceSecret map[string]string
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&serviceSecret); err != nil {
		return nil, err
	}

	privPEM := serviceSecret["private_key"]
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse private key PEM")
	}

	// Parse the private key
	privKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	ecdsaPrivKey, ok := privKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not ECDSA")
	}

	return ecdsaPrivKey, nil
}

func createJWTToken(serviceName string, privateKey *ecdsa.PrivateKey) (string, error) {
	// Create JWT claims
	claims := jwt.MapClaims{
		"service_name": serviceName,
		"exp":          time.Now().Add(time.Hour * 24).Unix(),
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	// Sign the token
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %v", err)
	}

	return tokenString, nil
}
