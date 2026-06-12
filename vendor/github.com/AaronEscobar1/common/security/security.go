package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// isProductionEnv indica si el backend corre en un entorno productivo según GO_ENV.
func isProductionEnv() bool {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("GO_ENV")))
	return env == "production" || env == "prod"
}

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	pemPublic  string
	onceKey    sync.Once
)

// InitRSAKeys genera de manera automática el par de claves RSA en memoria al arrancar la aplicación
func InitRSAKeys() error {
	var initErr error
	onceKey.Do(func() {
		slog.Info("Generando par de llaves RSA de 2048 bits en memoria para encriptación de tránsito...")
		
		privKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			initErr = fmt.Errorf("error al generar clave privada RSA: %w", err)
			return
		}

		privateKey = privKey
		publicKey = &privKey.PublicKey

		// Serializar clave pública a formato PEM
		pubASN1, err := x509.MarshalPKIXPublicKey(publicKey)
		if err != nil {
			initErr = fmt.Errorf("error al codificar clave pública x509: %w", err)
			return
		}

		pubBlock := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubASN1,
		}

		pemPublic = string(pem.EncodeToMemory(pubBlock))
		slog.Info("Par de llaves RSA generado exitosamente en memoria.")
	})
	return initErr
}

// GetPublicKeyPEM devuelve la clave pública RSA en formato PEM codificado.
func GetPublicKeyPEM() (string, error) {
	if pemPublic == "" {
		if err := InitRSAKeys(); err != nil {
			return "", err
		}
	}
	return pemPublic, nil
}

// DecryptPassword descifra una contraseña encriptada con RSA-OAEP (SHA-256) codificada en Base64.
func DecryptPassword(base64Ciphertext string) (string, error) {
	if privateKey == nil {
		if err := InitRSAKeys(); err != nil {
			return "", err
		}
	}

	if base64Ciphertext == "" {
		return "", errors.New("El texto cifrado está vacío")
	}

	if len(base64Ciphertext) < 300 {
		// SEGURIDAD: el "passthrough" de texto plano solo se permite fuera de producción
		// (Bruno/Postman/dev). En producción las credenciales DEBEN venir cifradas con RSA.
		if isProductionEnv() {
			slog.Warn("Credencial recibida sin cifrar en producción: rechazada por política de seguridad", "length", len(base64Ciphertext))
			return "", errors.New("Las credenciales deben enviarse cifradas (RSA) en este entorno")
		}
		slog.Debug("Contraseña recibida en texto plano. Omitiendo descifrado RSA (solo entornos de desarrollo).", "length", len(base64Ciphertext))
		return base64Ciphertext, nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(base64Ciphertext)
	if err != nil {
		slog.Warn("Fallo al decodificar texto cifrado desde Base64", "error", err)
		return "", errors.New("Formato de encriptación incorrecto (se esperaba Base64)")
	}

	rng := rand.Reader
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rng, privateKey, ciphertext, nil)
	if err != nil {
		slog.Warn("Fallo al desencriptar usando RSA-OAEP, intentando fallback PKCS#1 v1.5...", "error", err)
		plaintext, err = rsa.DecryptPKCS1v15(rng, privateKey, ciphertext)
		if err != nil {
			slog.Error("Fallo crítico al descifrar contraseña usando RSA", "error", err)
			return "", errors.New("Credenciales inválidas o error de descifrado")
		}
	}

	return string(plaintext), nil
}

// EncryptPassword cifra una contraseña en texto plano usando la clave pública RSA en memoria (RSA-OAEP con SHA-256) codificada en Base64.
func EncryptPassword(plaintext string) (string, error) {
	if publicKey == nil {
		if err := InitRSAKeys(); err != nil {
			return "", err
		}
	}

	rng := rand.Reader
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rng, publicKey, []byte(plaintext), nil)
	if err != nil {
		slog.Error("Fallo crítico al cifrar contraseña usando RSA", "error", err)
		return "", fmt.Errorf("error de cifrado: %w", err)
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// HashPassword hashea una contraseña utilizando bcrypt con coste por defecto.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compara una contraseña en texto plano con un hash bcrypt.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
