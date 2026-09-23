// Package crypto содержит логику работы с сертификатами: чтение, шифрование, расшифровка
package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// LoadPublicKey читает сертификат из файла
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	certBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading public key file: %w", err)
	}
	certPemBlock, _ := pem.Decode(certBytes)
	if certPemBlock == nil {
		return nil, errors.New("certificate not found")
	}
	certificate, err := x509.ParseCertificate(certPemBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error while parsing certificate: %w", err)
	}
	cert, ok := certificate.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not RSA")
	}
	return cert, nil
}

// LoadPrivateKey читает приватный ключ из файла
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading private key file: %w", err)
	}
	keyPemBlock, _ := pem.Decode(keyBytes)
	if keyPemBlock == nil {
		return nil, errors.New("private key not found")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(keyPemBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error while parsing key: %w", err)
	}
	return privateKey, nil
}

// Encrypt кодирует сообщение с использованием public key.
func Encrypt(key *rsa.PublicKey, data []byte) ([]byte, error) {
	// https://pkg.go.dev/crypto/rsa#EncryptOAEP
	// записывать нужно чанками
	maxChunk := key.Size() - (2 * sha256.Size) - 2
	if maxChunk < 1 {
		return nil, fmt.Errorf("RSA key too small for OAEP with SHA-256: got %d-byte key", key.Size())
	}
	numChunks := (len(data) + maxChunk - 1) / maxChunk
	if numChunks == 0 {
		numChunks = 1
	}
	result := make([]byte, 0, numChunks*key.Size())
	var i int
	for i = 0; i < numChunks-1; i++ {
		start, end := i*maxChunk, (i+1)*maxChunk
		encChunk, err := encryptChunk(key, data[start:end])
		if err != nil {
			return nil, fmt.Errorf("error while encrypting data: %w", err)
		}
		result = append(result, encChunk...)
	}
	// дописываем остатки
	start, end := i*maxChunk, len(data)
	lastChunk, err := encryptChunk(key, data[start:end])
	if err != nil {
		return nil, fmt.Errorf("error while encrypting rest of data: %w", err)
	}
	result = append(result, lastChunk...)
	return result, nil
}

func encryptChunk(key *rsa.PublicKey, data []byte) ([]byte, error) {
	encryptedData, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, key, data, nil)
	if err != nil {
		return nil, fmt.Errorf("error while encrypting chunk: %w", err)
	}
	return encryptedData, nil
}

// Decrypt декодирует сообщение с использованием private key.
func Decrypt(key *rsa.PrivateKey, data []byte) ([]byte, error) {
	result := make([]byte, 0, len(data))
	chunkSize := key.Size()
	if len(data)%chunkSize != 0 {
		return nil, errors.New("cannot parse encrypted data, invalid key size")
	}
	for i := 0; ((i + 1) * chunkSize) <= len(data); i++ {
		start, end := i*chunkSize, (i+1)*chunkSize
		decryptedData, err := rsa.DecryptOAEP(sha256.New(), nil, key, data[start:end], nil)
		if err != nil {
			return nil, fmt.Errorf("error while decrypting data: %w", err)
		}
		result = append(result, decryptedData...)
	}
	return result, nil
}
