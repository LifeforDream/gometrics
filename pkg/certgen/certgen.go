// Package certgen содержит логику генерации сертификата x509, не касаясь I/O
package certgen

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"
)

// GenerateKeyPair генерирует пару ключей (сертификат и приватный ключ)
// и сохраняет их в файлы cert.pem и private.pem в указанной директории.
//
// Функция принимает идентификатор сертификата, путь к директории для сохранения файлов и количество бит для генерации ключа.
func GenerateKeyPair(certID int64, bits int) ([]byte, []byte, error) {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(certID),
		Subject: pkix.Name{
			Organization: []string{"Yandex.Praktikum"},
			Country:      []string{"RU"},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	var certBuf bytes.Buffer
	err = pem.Encode(&certBuf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return nil, nil, err
	}

	var privateKeyBuf bytes.Buffer
	err = pem.Encode(&privateKeyBuf, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err != nil {
		return nil, nil, err
	}

	return certBuf.Bytes(), privateKeyBuf.Bytes(), nil
}
