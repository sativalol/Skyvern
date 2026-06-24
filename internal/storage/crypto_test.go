package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strings"
	"testing"

	"golang.org/x/crypto/pbkdf2"
)

func TestEncryptDecrypt(t *testing.T) {
	os.Setenv(cryptEnvVar, "test_secret_key_1234567890_abc")
	defer os.Unsetenv(cryptEnvVar)

	aesKeyV1 = nil
	aesKeyV2 = nil

	plaintext := "hello world secret message!"
	ciphertext, err := encrypt(plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	if !strings.HasPrefix(ciphertext, cryptPrefixV2) {
		t.Errorf("ciphertext lacks expected V2 prefix: %s", ciphertext)
	}

	decrypted, err := decrypt(ciphertext)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypted message %q doesn't match original %q", decrypted, plaintext)
	}
}

func TestDecryptV1Compatibility(t *testing.T) {
	os.Setenv(cryptEnvVar, "test_secret_key_1234567890_abc")
	defer os.Unsetenv(cryptEnvVar)

	aesKeyV1 = nil
	aesKeyV2 = nil

	// Manually generate a V1 ciphertext
	salt := []byte("skyvern_kdf_salt_1337_!")
	keyV1 := pbkdf2.Key([]byte("test_secret_key_1234567890_abc"), salt, 10000, 32, sha256.New)

	plaintext := "backwards compatible test"
	block, err := aes.NewCipher(keyV1)
	if err != nil {
		t.Fatalf("failed to create cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("failed to create gcm: %v", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		t.Fatalf("failed to read nonce: %v", err)
	}
	cipherbytes := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	ciphertextV1 := cryptPrefixV1 + hex.EncodeToString(cipherbytes)

	// Decrypt it using the new decrypt function
	decrypted, err := decrypt(ciphertextV1)
	if err != nil {
		t.Fatalf("failed to decrypt V1 payload: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypted V1 message %q doesn't match original %q", decrypted, plaintext)
	}
}

func TestDecryptTampered(t *testing.T) {
	os.Setenv(cryptEnvVar, "test_secret_key_1234567890_abc")
	defer os.Unsetenv(cryptEnvVar)

	aesKeyV1 = nil
	aesKeyV2 = nil

	ciphertext, err := encrypt("secret data")
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	parts := strings.Split(ciphertext, cryptPrefixV2)
	if len(parts) < 2 {
		t.Fatalf("invalid ciphertext format: %s", ciphertext)
	}
	tampered := cryptPrefixV2 + parts[1][:5] + "f" + parts[1][6:]

	_, err = decrypt(tampered)
	if err == nil {
		t.Error("expected error when decrypting tampered data, got nil")
	}
}
