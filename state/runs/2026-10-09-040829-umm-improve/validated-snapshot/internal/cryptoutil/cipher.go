package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strings"
)

type cipherKey struct {
	id   string
	aead cipher.AEAD
}

type Cipher struct {
	primary cipherKey
	keys    []cipherKey
}

func New(key []byte) (*Cipher, error) {
	return NewWithPrevious(key)
}

func NewWithPrevious(primary []byte, previous ...[]byte) (*Cipher, error) {
	all := append([][]byte{primary}, previous...)
	keys := make([]cipherKey, 0, len(all))
	seen := map[string]bool{}
	for _, raw := range all {
		block, err := aes.NewCipher(raw)
		if err != nil {
			return nil, err
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(raw)
		id := hex.EncodeToString(digest[:6])
		if seen[id] {
			continue
		}
		seen[id] = true
		keys = append(keys, cipherKey{id: id, aead: aead})
	}
	if len(keys) == 0 {
		return nil, errors.New("at least one encryption key is required")
	}
	return &Cipher{primary: keys[0], keys: keys}, nil
}

func (c *Cipher) Encrypt(plain string) (string, error) {
	nonce := make([]byte, c.primary.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	aad := []byte("umm:v2:" + c.primary.id)
	sealed := c.primary.aead.Seal(nonce, nonce, []byte(plain), aad)
	return "v2." + c.primary.id + "." + base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (c *Cipher) Decrypt(encoded string) (string, error) {
	if strings.HasPrefix(encoded, "v2.") {
		parts := strings.SplitN(encoded, ".", 3)
		if len(parts) != 3 {
			return "", errors.New("invalid encrypted value")
		}
		for _, key := range c.keys {
			if key.id == parts[1] {
				return decryptWith(key.aead, parts[2], []byte("umm:v2:"+key.id))
			}
		}
		return "", errors.New("encrypted value requires an unavailable key")
	}
	for _, key := range c.keys {
		plain, err := decryptWith(key.aead, encoded, []byte("umm:v1"))
		if err == nil {
			return plain, nil
		}
	}
	return "", errors.New("unable to decrypt value")
}

func decryptWith(aead cipher.AEAD, encoded string, aad []byte) (string, error) {
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(data) < aead.NonceSize() {
		return "", errors.New("invalid encrypted value")
	}
	nonce, data := data[:aead.NonceSize()], data[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, data, aad)
	if err != nil {
		return "", errors.New("unable to decrypt value")
	}
	return string(plain), nil
}

func (c *Cipher) KeyID() string { return c.primary.id }

func (c *Cipher) FallbackKeyCount() int { return max(0, len(c.keys)-1) }

func (c *Cipher) NeedsRotation(encoded string) bool {
	return !strings.HasPrefix(encoded, "v2."+c.primary.id+".")
}

func Digest(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}
