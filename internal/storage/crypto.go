package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"skyvern/internal/config"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	cryptEnvVar   = "SKYVERN_CRYPT_KEY"
	cryptPrefixV1 = "$crypt$gcm$"
	cryptPrefixV2 = "$crypt$gcm$v2$"
)

var (
	aesKeyV1 []byte
	aesKeyV2 []byte
)

func InitCrypto() error {
	if aesKeyV2 != nil {
		return nil
	}
	keyStr := os.Getenv(cryptEnvVar)
	if keyStr == "" {
		cfg := config.GetTuiCfg()
		keyStr = cfg.CryptKey
		if keyStr == "" {
			b := make([]byte, 32)
			if _, err := rand.Read(b); err != nil {
				return fmt.Errorf("failed to generate random key: %w", err)
			}
			keyStr = hex.EncodeToString(b)
			cfg.CryptKey = keyStr
			if err := config.SaveTuiCfg(cfg); err != nil {
				return fmt.Errorf("failed to save generated key: %w", err)
			}
			fmt.Println("[+] No encryption key found. Generated a new random master key and saved to tui_config.json.")
		}
	}
	salt := []byte("skyvern_kdf_salt_1337_!")
	aesKeyV1 = pbkdf2.Key([]byte(keyStr), salt, 10000, 32, sha256.New)
	aesKeyV2 = pbkdf2.Key([]byte(keyStr), salt, 600000, 32, sha256.New)
	return nil
}

func getAESKeyV1() []byte {
	if aesKeyV1 == nil {
		if err := InitCrypto(); err != nil {
			panic(err)
		}
	}
	return aesKeyV1
}

func getAESKeyV2() []byte {
	if aesKeyV2 == nil {
		if err := InitCrypto(); err != nil {
			panic(err)
		}
	}
	return aesKeyV2
}

func encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(getAESKeyV2())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return cryptPrefixV2 + hex.EncodeToString(ciphertext), nil
}

func decrypt(ciphertextStr string) (string, error) {
	if ciphertextStr == "" {
		return "", nil
	}
	var key []byte
	var payload string
	if strings.HasPrefix(ciphertextStr, cryptPrefixV2) {
		payload = strings.TrimPrefix(ciphertextStr, cryptPrefixV2)
		key = getAESKeyV2()
	} else if strings.HasPrefix(ciphertextStr, cryptPrefixV1) {
		payload = strings.TrimPrefix(ciphertextStr, cryptPrefixV1)
		key = getAESKeyV1()
	} else {
		return ciphertextStr, nil
	}
	data, err := hex.DecodeString(payload)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, cipherbytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherbytes, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
