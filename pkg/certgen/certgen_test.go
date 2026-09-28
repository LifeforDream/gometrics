package certgen

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCertID = 1

func TestGenerateKeyPair(t *testing.T) {
	for _, bits := range []int{1024, 2048} {
		t.Run(fmt.Sprintf("%d bits", bits), func(t *testing.T) {
			certPEM, privPEM, err := GenerateKeyPair(testCertID, bits)
			require.NoError(t, err)
			require.NotEmpty(t, certPEM)
			require.NotEmpty(t, privPEM)

			certBlock, rest := pem.Decode(certPEM)
			require.NotNil(t, certBlock)
			assert.Empty(t, rest)
			assert.Equal(t, "CERTIFICATE", certBlock.Type)

			cert, err := x509.ParseCertificate(certBlock.Bytes)
			require.NoError(t, err)
			pub, ok := cert.PublicKey.(*rsa.PublicKey)
			require.True(t, ok)
			assert.Equal(t, bits, pub.N.BitLen())

			keyBlock, rest := pem.Decode(privPEM)
			require.NotNil(t, keyBlock)
			assert.Empty(t, rest)
			assert.Equal(t, "RSA PRIVATE KEY", keyBlock.Type)

			priv, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
			require.NoError(t, err)
			assert.Equal(t, bits, priv.N.BitLen())

			// ключи действительно образуют пару
			plaintext := []byte("round trip check")
			ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, plaintext, nil)
			require.NoError(t, err)
			decrypted, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, ciphertext, nil)
			require.NoError(t, err)
			assert.Equal(t, plaintext, decrypted)
		})
	}
}

func TestGenerateKeyPairProducesFreshKeysEachCall(t *testing.T) {
	certA, privA, err := GenerateKeyPair(testCertID, 1024)
	require.NoError(t, err)
	certB, privB, err := GenerateKeyPair(testCertID, 1024)
	require.NoError(t, err)

	assert.NotEqual(t, certA, certB)
	assert.NotEqual(t, privA, privB)
}
