package banking

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
)

// Vault encrypts credentials with AES-256-GCM and authenticated record context.
type Vault struct {
	active string
	keys   map[string]cipher.AEAD
}

// NewVault builds a key ring; old keys remain available for rotation and recovery.
func NewVault(c Config) (*Vault, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	v := &Vault{active: c.ActiveKeyID, keys: map[string]cipher.AEAD{}}
	for id, key := range c.Keys {
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}
		v.keys[id] = aead
	}
	return v, nil
}

// Seal binds each encrypted credential to its household, record, and purpose.
func (v *Vault) Seal(value, context string) (string, error) {
	aead := v.keys[v.active]
	if aead == nil {
		return "", errors.New("encryption unavailable")
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, []byte(value), []byte(context))
	return v.active + "." + base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Open authenticates ciphertext and its record context before returning a secret.
func (v *Vault) Open(value, context string) (string, error) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid encrypted credential")
	}
	aead := v.keys[parts[0]]
	if aead == nil {
		return "", errors.New("encryption key unavailable")
	}
	sealed, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(sealed) < aead.NonceSize()+aead.Overhead() {
		return "", errors.New("invalid encrypted credential")
	}
	data, err := aead.Open(nil, sealed[:aead.NonceSize()], sealed[aead.NonceSize():], []byte(context))
	if err != nil {
		return "", errors.New("credential authentication failed")
	}
	return string(data), nil
}
