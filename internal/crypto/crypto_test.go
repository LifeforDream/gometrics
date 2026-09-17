package crypto

import (
	"crypto/rsa"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LifeforDream/gometrics/pkg/certgen"
)

const (
	testKeyBits = 1024
	testCertID  = 1
)

func generateTestKeyFiles(t *testing.T, bits int) (string, string) {
	t.Helper()
	certPEM, privPEM, err := certgen.GenerateKeyPair(testCertID, bits)
	require.NoError(t, err)

	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	privPath := filepath.Join(dir, "private.pem")
	require.NoError(t, os.WriteFile(certPath, certPEM, 0o600))
	require.NoError(t, os.WriteFile(privPath, privPEM, 0o600))
	return certPath, privPath
}

// generateTestKeys генерирует пару ключей через certgen.GenerateKeyPair
// и загружает их обратно, возвращая готовые к использованию объекты.
func generateTestKeys(t *testing.T, bits int) (*rsa.PublicKey, *rsa.PrivateKey) {
	t.Helper()
	certPath, privPath := generateTestKeyFiles(t, bits)

	pub, err := LoadPublicKey(certPath)
	require.NoError(t, err)
	priv, err := LoadPrivateKey(privPath)
	require.NoError(t, err)

	return pub, priv
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	pub, priv := generateTestKeys(t, testKeyBits)

	// maxChunk - максимальный размер одного RSA-OAEP блока для данного
	// размера ключа (SHA-256 в качестве хэша OAEP).
	maxChunk := pub.Size() - 2*sha256.Size - 2
	require.Greater(t, maxChunk, 0)

	tests := []struct {
		name string
		size int
	}{
		{"empty payload", 0},
		{"one byte", 1},
		{"exactly one OAEP block", maxChunk},
		{"one byte over a single block", maxChunk + 1},
		{"several blocks with remainder", maxChunk*3 + 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plaintext := make([]byte, tt.size)
			for i := range plaintext {
				plaintext[i] = byte(i % 251)
			}

			ciphertext, err := Encrypt(pub, plaintext)
			require.NoError(t, err)
			if tt.size > 0 {
				assert.NotEqual(t, plaintext, ciphertext)
			}

			decrypted, err := Decrypt(priv, ciphertext)
			require.NoError(t, err)
			assert.Equal(t, plaintext, decrypted)
		})
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	pubA, _ := generateTestKeys(t, testKeyBits)
	_, privB := generateTestKeys(t, testKeyBits)

	ciphertext, err := Encrypt(pubA, []byte("secret payload"))
	require.NoError(t, err)

	_, err = Decrypt(privB, ciphertext)
	assert.Error(t, err)
}

func TestDecryptMalformedCiphertext(t *testing.T) {
	_, priv := generateTestKeys(t, testKeyBits)

	_, err := Decrypt(priv, []byte("not a valid ciphertext block, wrong length"))
	assert.Error(t, err)
}

func TestLoadPublicKey(t *testing.T) {
	certPath, privPath := generateTestKeyFiles(t, testKeyBits)

	t.Run("valid certificate", func(t *testing.T) {
		pub, err := LoadPublicKey(certPath)
		require.NoError(t, err)
		assert.NotNil(t, pub)
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := LoadPublicKey(filepath.Join(t.TempDir(), "missing.pem"))
		assert.Error(t, err)
	})

	t.Run("not a PEM file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "garbage.pem")
		require.NoError(t, os.WriteFile(path, []byte("not pem data"), 0o600))
		_, err := LoadPublicKey(path)
		assert.Error(t, err)
	})

	t.Run("wrong PEM block type (private key given)", func(t *testing.T) {
		_, err := LoadPublicKey(privPath)
		assert.Error(t, err)
	})
}

func TestLoadPrivateKey(t *testing.T) {
	certPath, privPath := generateTestKeyFiles(t, testKeyBits)

	t.Run("valid private key", func(t *testing.T) {
		priv, err := LoadPrivateKey(privPath)
		require.NoError(t, err)
		assert.NotNil(t, priv)
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := LoadPrivateKey(filepath.Join(t.TempDir(), "missing.pem"))
		assert.Error(t, err)
	})

	t.Run("not a PEM file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "garbage.pem")
		require.NoError(t, os.WriteFile(path, []byte("not pem data"), 0o600))
		_, err := LoadPrivateKey(path)
		assert.Error(t, err)
	})

	t.Run("wrong PEM block type (certificate given)", func(t *testing.T) {
		_, err := LoadPrivateKey(certPath)
		assert.Error(t, err)
	})
}
