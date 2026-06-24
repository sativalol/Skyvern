package utility
import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"skyvern/internal/manager"
)
func encryptBackup(data []byte, pass string) ([]byte, error) {
	key := sha256.Sum256([]byte(pass))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}
func decryptBackup(data []byte, pass string) ([]byte, error) {
	key := sha256.Sum256([]byte(pass))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(data) < ns {
		return nil, fmt.Errorf("invalid encrypted payload size")
	}
	nonce, ciphertext := data[:ns], data[ns:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (incorrect password or corrupted payload): %w", err)
	}
	return plaintext, nil
}
func getSecret(ctx *manager.CommandContext, pass string) string {
	if pass != "" {
		return pass
	}
	if ctx.Session != nil && ctx.Session.Token != "" {
		return ctx.Session.Token
	}
	return "skyvern-backup-secret-key-default-fallback-32b"
}