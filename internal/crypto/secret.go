// Package crypto provides AES-GCM encryption and decryption for sensitive
// configuration values (auth keys, API keys, operator passwords).
// Secrets are encrypted with a key derived from a machine-specific salt.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/scrypt"
)

// Envelope holds the encrypted payload and metadata for AES-GCM sealed secrets.
type Envelope struct {
	Mode   string `json:"mode"`
	Salt   string `json:"salt"`
	Nonce  string `json:"nonce"`
	Cipher string `json:"cipher"`
}

// Encrypt encrypts plaintext using AES-GCM with a scrypt-derived key bound to the machine or a password.
func Encrypt(plain, password, machineContext string) (string, error) {
	key, salt, mode, err := deriveKey(password, machineContext)
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
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	enc := gcm.Seal(nil, nonce, []byte(plain), nil)
	env := Envelope{
		Mode:   mode,
		Salt:   base64.StdEncoding.EncodeToString(salt),
		Nonce:  base64.StdEncoding.EncodeToString(nonce),
		Cipher: base64.StdEncoding.EncodeToString(enc),
	}
	b, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// Decrypt decodes and decrypts a base64-encoded Envelope back to plaintext.
func Decrypt(encoded, password, machineContext string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", err
	}
	salt, err := base64.StdEncoding.DecodeString(env.Salt)
	if err != nil {
		return "", err
	}
	nonce, err := base64.StdEncoding.DecodeString(env.Nonce)
	if err != nil {
		return "", err
	}
	cipherBytes, err := base64.StdEncoding.DecodeString(env.Cipher)
	if err != nil {
		return "", err
	}
	key, _, _, err := deriveKeyWithSalt(password, machineContext, salt, env.Mode)
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
	out, err := gcm.Open(nil, nonce, cipherBytes, nil)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func deriveKey(password, machineContext string) ([]byte, []byte, string, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, nil, "", err
	}
	mode := "machine"
	if strings.TrimSpace(password) != "" {
		mode = "password"
	}
	key, _, _, err := deriveKeyWithSalt(password, machineContext, salt, mode)
	if err != nil {
		return nil, nil, "", err
	}
	return key, salt, mode, nil
}

func deriveKeyWithSalt(password, machineContext string, salt []byte, mode string) ([]byte, []byte, string, error) {
	base := password
	if mode == "machine" {
		base = machineContext
	}
	if strings.TrimSpace(base) == "" {
		return nil, nil, "", fmt.Errorf("empty key material")
	}
	combined := sha256.Sum256([]byte(base))
	key, err := scrypt.Key(combined[:], salt, 1<<15, 8, 1, 32)
	if err != nil {
		return nil, nil, "", err
	}
	return key, salt, mode, nil
}
