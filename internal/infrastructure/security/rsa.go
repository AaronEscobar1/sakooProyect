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
	"sync"
)

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

	// Inteligente Fallback para Desarrollo y Pruebas Manuales (Bruno/Postman):
	// Un texto cifrado RSA-2048 codificado en Base64 siempre tiene exactamente 344 caracteres.
	// Si la longitud es menor a 300 caracteres, asumimos que es una contraseña en texto plano.
	if len(base64Ciphertext) < 300 {
		slog.Debug("Contraseña recibida en texto plano. Omitiendo descifrado RSA (Compatible con Bruno/Postman).", "length", len(base64Ciphertext))
		return base64Ciphertext, nil
	}

	// 1. Decodificar desde Base64
	ciphertext, err := base64.StdEncoding.DecodeString(base64Ciphertext)
	if err != nil {
		slog.Warn("Fallo al decodificar texto cifrado desde Base64", "error", err)
		return "", errors.New("Formato de encriptación incorrecto (se esperaba Base64)")
	}

	// 2. Desencriptar usando RSA-OAEP con hash SHA-256
	rng := rand.Reader
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rng, privateKey, ciphertext, nil)
	if err != nil {
		// Intentar fallback a RSA-PKCS1v15 para mayor compatibilidad con algunas librerías frontend sencillas
		slog.Warn("Fallo al desencriptar usando RSA-OAEP, intentando fallback PKCS#1 v1.5...", "error", err)
		plaintext, err = rsa.DecryptPKCS1v15(rng, privateKey, ciphertext)
		if err != nil {
			slog.Error("Fallo crítico al descifrar contraseña usando RSA", "error", err)
			return "", errors.New("Credenciales inválidas o error de descifrado")
		}
	}

	return string(plaintext), nil
}
