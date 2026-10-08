package bootstrap

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hkjang/invenqor/server/internal/durablefs"
)

const (
	keyFileName       = "master.key"
	bootstrapFileName = "bootstrap.enc"
	keySize           = 32
)

type Values struct {
	PostgresDSN          string `json:"postgres_dsn,omitempty"`
	SQLitePath           string `json:"sqlite_path,omitempty"`
	KeycloakClientSecret string `json:"keycloak_client_secret,omitempty"`
}

type encryptedEnvelope struct {
	Version    int    `json:"version"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

type Store struct {
	root          string
	keyPath       string
	bootstrapPath string
	key           []byte
}

func Open(root string) (*Store, error) {
	return OpenWithKey(root, "")
}

func OpenWithKey(root string, externalKeyPath string) (*Store, error) {
	if err := secureDirectory(root); err != nil {
		return nil, err
	}
	store := &Store{
		root:          root,
		keyPath:       filepath.Join(root, keyFileName),
		bootstrapPath: filepath.Join(root, bootstrapFileName),
	}
	var key []byte
	var err error
	if externalKeyPath != "" {
		store.keyPath = externalKeyPath
		key, err = os.ReadFile(externalKeyPath)
		if err == nil && len(key) != keySize {
			err = fmt.Errorf("external master key must be %d bytes", keySize)
		}
	} else {
		key, err = loadOrCreateKey(store.keyPath)
	}
	if err != nil {
		return nil, fmt.Errorf("load master key: %w", err)
	}
	store.key = key
	return store, nil
}

func (s *Store) Exists() bool {
	_, err := os.Stat(s.bootstrapPath)
	return err == nil
}

func (s *Store) Load() (Values, error) {
	bytes, err := os.ReadFile(s.bootstrapPath)
	if errors.Is(err, os.ErrNotExist) {
		return Values{}, nil
	}
	if err != nil {
		return Values{}, fmt.Errorf("read encrypted bootstrap settings: %w", err)
	}
	var envelope encryptedEnvelope
	if err := json.Unmarshal(bytes, &envelope); err != nil {
		// Same reasoning as a damaged master key: the values are unreachable, and
		// starting without them loses configured secrets silently.
		return Values{}, fmt.Errorf(
			"%s is not readable as a sealed envelope, so the settings it holds "+
				"cannot be recovered. Restore it from a backup, or delete it: the "+
				"Server will start without those values, and every sealed setting - "+
				"including the Keycloak client secret - must be entered again",
			s.bootstrapPath,
		)
	}
	if envelope.Version != 1 {
		return Values{}, fmt.Errorf("unsupported bootstrap envelope version %d", envelope.Version)
	}
	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return Values{}, errors.New("decode bootstrap nonce")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return Values{}, errors.New("decode bootstrap ciphertext")
	}
	aead, err := newAEAD(s.key)
	if err != nil {
		return Values{}, err
	}
	plaintext, err := aead.Open(nil, nonce, ciphertext, []byte("invenqor-bootstrap-v1"))
	if err != nil {
		return Values{}, errors.New("decrypt bootstrap settings")
	}
	var values Values
	if err := json.Unmarshal(plaintext, &values); err != nil {
		return Values{}, errors.New("decode bootstrap settings")
	}
	return values, nil
}

func (s *Store) Save(values Values) error {
	plaintext, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("encode bootstrap settings: %w", err)
	}
	aead, err := newAEAD(s.key)
	if err != nil {
		return err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("generate bootstrap nonce: %w", err)
	}
	ciphertext := aead.Seal(nil, nonce, plaintext, []byte("invenqor-bootstrap-v1"))
	envelope, err := json.Marshal(encryptedEnvelope{
		Version:    1,
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	})
	if err != nil {
		return fmt.Errorf("encode encrypted bootstrap envelope: %w", err)
	}
	envelope = append(envelope, '\n')
	if err := atomicWrite(s.bootstrapPath, envelope, 0o600); err != nil {
		return fmt.Errorf("store encrypted bootstrap settings: %w", err)
	}
	return nil
}

// SealString encrypts a runtime secret with the bootstrap master key. The
// purpose is authenticated as associated data so ciphertext cannot be moved
// between secret fields.
func (s *Store) SealString(purpose, plaintext string) (string, error) {
	if purpose == "" {
		return "", errors.New("encryption purpose is required")
	}
	aead, err := newAEAD(s.key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate secret nonce: %w", err)
	}
	ciphertext := aead.Seal(nil, nonce, []byte(plaintext), []byte(purpose))
	return "v1." +
		base64.RawURLEncoding.EncodeToString(nonce) + "." +
		base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func (s *Store) OpenString(purpose, encoded string) (string, error) {
	parts := strings.Split(encoded, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return "", errors.New("invalid encrypted secret envelope")
	}
	nonce, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errors.New("decode secret nonce")
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", errors.New("decode secret ciphertext")
	}
	aead, err := newAEAD(s.key)
	if err != nil {
		return "", err
	}
	plaintext, err := aead.Open(nil, nonce, ciphertext, []byte(purpose))
	if err != nil {
		return "", errors.New("decrypt runtime secret")
	}
	return string(plaintext), nil
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("initialize bootstrap encryption: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("initialize bootstrap AEAD: %w", err)
	}
	return aead, nil
}

func loadOrCreateKey(path string) ([]byte, error) {
	key, err := os.ReadFile(path)
	if err == nil {
		if len(key) != keySize {
			// Refusing to start is right: without the original key the sealed
			// settings - the Keycloak client secret among them - cannot be
			// decrypted, and starting anyway would make SSO vanish from the login
			// page with nothing to explain it. But the operator gets this line and
			// a container that restarts, so it has to say what to do about it.
			return nil, fmt.Errorf(
				"%s is %d bytes, not %d, so the sealed settings in this state "+
					"directory cannot be decrypted. Restore that file from a backup, "+
					"or point INVENQOR_MASTER_KEY_FILE at a copy of the original key. "+
					"If neither is available, delete %s and bootstrap.enc beside it: "+
					"the Server will start with a new key, and every sealed setting - "+
					"including the Keycloak client secret - must be entered again",
				path, len(key), keySize, path,
			)
		}
		if err := os.Chmod(path, 0o600); err != nil {
			return nil, fmt.Errorf("secure master key: %w", err)
		}
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read master key: %w", err)
	}
	key = make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate master key: %w", err)
	}
	if err := atomicWrite(path, key, 0o600); err != nil {
		return nil, fmt.Errorf("store master key: %w", err)
	}
	return key, nil
}

func secureDirectory(path string) error {
	if err := durablefs.EnsurePrivateDirectory(path); err != nil {
		return fmt.Errorf("secure bootstrap state directory: %w", err)
	}
	return nil
}

func atomicWrite(path string, bytes []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	file, err := os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(bytes); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	return durablefs.SyncDirectory(directory)
}
