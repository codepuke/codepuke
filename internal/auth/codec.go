// Package auth is the Authentik OIDC client and the encrypted cookie
// sessions behind /admin. There is no session table: the cookie is the
// session, sealed with XChaCha20-Poly1305.
package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// ErrBadToken is returned by Open for any token that does not decrypt: too
// short, tampered, or sealed with a different key.
var ErrBadToken = errors.New("bad token")

// Codec seals small JSON payloads into opaque cookie-safe strings.
type Codec struct {
	key []byte
}

// NewCodec builds a Codec around a 32-byte key.
func NewCodec(key []byte) (*Codec, error) {
	if len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("session key must be %d bytes, got %d", chacha20poly1305.KeySize, len(key))
	}
	return &Codec{key: append([]byte(nil), key...)}, nil
}

// Seal encrypts v as JSON. The random nonce is prepended, so every call
// produces a distinct token.
func (c *Codec) Seal(v any) (string, error) {
	plain, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	aead, err := chacha20poly1305.NewX(c.key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize(), aead.NonceSize()+len(plain)+aead.Overhead())
	if err := fillRandom(nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, plain, nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Open decrypts a Seal token into v.
func (c *Codec) Open(token string, v any) error {
	sealed, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return ErrBadToken
	}
	aead, err := chacha20poly1305.NewX(c.key)
	if err != nil {
		return err
	}
	if len(sealed) < aead.NonceSize() {
		return ErrBadToken
	}
	plain, err := aead.Open(nil, sealed[:aead.NonceSize()], sealed[aead.NonceSize():], nil)
	if err != nil {
		return ErrBadToken
	}
	if err := json.Unmarshal(plain, v); err != nil {
		return ErrBadToken
	}
	return nil
}
