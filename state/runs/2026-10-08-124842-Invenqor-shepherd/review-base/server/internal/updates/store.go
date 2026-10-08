package updates

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hkjang/invenqor/server/internal/durablefs"
)

type Manifest struct {
	Version      string `json:"version"`
	Channel      string `json:"channel"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	SHA256       string `json:"sha256"`
	// Signature remains the artifact-only Ed25519 signature understood by
	// pre-v2 Agents. ManifestSignature authenticates the canonical v2 metadata.
	Signature         string `json:"signature"`
	ManifestSignature string `json:"manifest_signature,omitempty"`
	// SignatureScheme and SignatureVersion are always explicit on the wire.
	// Releases written before manifest signing existed are normalized to the
	// legacy artifact-only Ed25519 contract when they are read.
	SignatureScheme  string `json:"signature_scheme"`
	SignatureVersion int    `json:"signature_version"`
	DownloadURL      string `json:"download_url"`
	Size             int64  `json:"size"`
	Rollout          int    `json:"rollout_percent"`
	// AllowDowngrade lets an operator move a fleet back to an earlier build. A
	// rollback is the one case where "not newer" is the point, and it stays safe
	// because the artifact is still signed and hash-checked.
	AllowDowngrade bool `json:"allow_downgrade,omitempty"`
	// Notes travels with the release so the console can show what changed.
	Notes string `json:"notes,omitempty"`
	// PublishedAt and PublishedBy make a release auditable from the store alone.
	PublishedAt time.Time `json:"published_at"`
	PublishedBy string    `json:"published_by,omitempty"`
	// SignatureVerified records whether this server could check the signature at
	// publish time, so the console never implies a check that did not happen.
	SignatureVerified bool `json:"signature_verified"`
}

// Release is the operator's view of one published build.
type Release struct {
	Manifest
	Base string `json:"base"`
}

type Store struct {
	root      string
	mu        sync.RWMutex
	publicKey ed25519.PublicKey
	now       func() time.Time
}

func Open(root string) (*Store, error) {
	if err := durablefs.EnsurePrivateDirectory(root); err != nil {
		return nil, err
	}
	return &Store{root: root, now: func() time.Time { return time.Now().UTC() }}, nil
}

func (s *Store) Publish(manifest Manifest, source io.Reader) (Manifest, error) {
	if !safeVersion(manifest.Version) {
		return Manifest{}, errors.New(
			"the version must be three numbers, for example 0.2.7",
		)
	}
	if manifest.Channel != "stable" && manifest.Channel != "beta" {
		return Manifest{}, errors.New("the channel must be stable or beta")
	}
	if manifest.OS != "linux" && manifest.OS != "windows" {
		return Manifest{}, errors.New("the os must be linux or windows")
	}
	if manifest.Architecture != "x86_64" && manifest.Architecture != "aarch64" {
		return Manifest{}, errors.New(
			"the architecture must be x86_64 or aarch64",
		)
	}
	// A release is identified by version, os and architecture together, so a
	// Windows build and a Linux build of the same version can coexist. Publishing
	// a Windows artifact where a Linux one is expected would otherwise be
	// indistinguishable until it failed its self-test on every host.
	if manifest.OS == "windows" && manifest.Architecture != "x86_64" {
		return Manifest{}, errors.New(
			"windows releases are published for x86_64",
		)
	}
	if manifest.Rollout < 0 || manifest.Rollout > 100 {
		return Manifest{}, errors.New("the rollout percent must be 0 to 100")
	}
	if manifest.SignatureScheme != SignatureSchemeEd25519 ||
		manifest.SignatureVersion != SignatureVersionV2 {
		return Manifest{}, fmt.Errorf(
			"new releases require signature_scheme %q and signature_version %d",
			SignatureSchemeEd25519, SignatureVersionV2,
		)
	}
	legacySignature, err := DecodeSignature(manifest.Signature)
	if err != nil {
		return Manifest{}, fmt.Errorf("decode legacy artifact signature: %w", err)
	}
	manifestSignature, err := DecodeSignature(manifest.ManifestSignature)
	if err != nil {
		return Manifest{}, fmt.Errorf("decode v2 manifest signature: %w", err)
	}
	manifest.Signature = base64.StdEncoding.EncodeToString(legacySignature)
	manifest.ManifestSignature = base64.StdEncoding.EncodeToString(manifestSignature)

	base := manifest.Version + "-" + manifest.OS + "-" + manifest.Architecture
	// Read the artifact into a temporary file first: its exact size and digest
	// are part of the v2 signature contract, and a rejected publication must
	// leave nothing behind.
	staged, hash, size, err := s.stageArtifact(base, source)
	if err != nil {
		return Manifest{}, err
	}
	defer os.Remove(staged)

	if manifest.Size != 0 && manifest.Size != size {
		return Manifest{}, fmt.Errorf(
			"the signature bundle size does not match the artifact: %w",
			ErrSignatureRejected,
		)
	}
	if manifest.SHA256 != "" && manifest.SHA256 != hash {
		return Manifest{}, fmt.Errorf(
			"the signature bundle sha256 does not match the artifact: %w",
			ErrSignatureRejected,
		)
	}
	manifest.SHA256 = hash
	manifest.Size = size
	signedMessage, err := SignatureMessageV2(manifest)
	if err != nil {
		return Manifest{}, err
	}
	if s.SigningKeyConfigured() {
		artifactBytes, readErr := os.ReadFile(staged)
		if readErr != nil {
			return Manifest{}, readErr
		}
		if err := s.verifySignature(legacySignature, artifactBytes); err != nil {
			return Manifest{}, fmt.Errorf("legacy artifact signature: %w", err)
		}
		if err := s.verifySignature(manifestSignature, signedMessage); err != nil {
			return Manifest{}, fmt.Errorf("v2 manifest signature: %w", err)
		}
		manifest.SignatureVerified = true
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	artifact := filepath.Join(s.root, base+".bin")
	manifestPath := filepath.Join(s.root, base+".json")
	manifest.DownloadURL = "/v1/agent/updates/" + base + "/artifact"
	if manifest.PublishedAt.IsZero() {
		manifest.PublishedAt = s.now()
	}
	bytes, _ := json.MarshalIndent(manifest, "", "  ")
	err = s.withFileLock(func() error {
		artifactExists := fileExists(artifact)
		manifestExists := fileExists(manifestPath)
		if artifactExists && manifestExists {
			return fmt.Errorf(
				"%s for %s is already published; publish a new version or adjust its rollout",
				manifest.Version, manifest.Architecture,
			)
		}
		// A process can crash after committing the artifact but before the
		// manifest. Conversely, an externally damaged store can contain only a
		// manifest. Recover either half-commit while holding the shared lock.
		if artifactExists {
			if removeErr := os.Remove(artifact); removeErr != nil {
				return removeErr
			}
		}
		if manifestExists {
			if removeErr := os.Remove(manifestPath); removeErr != nil {
				return removeErr
			}
		}
		if artifactExists || manifestExists {
			if syncErr := durablefs.SyncDirectory(s.root); syncErr != nil {
				return syncErr
			}
		}
		if linkErr := os.Link(staged, artifact); linkErr != nil {
			return linkErr
		}
		if syncErr := durablefs.SyncDirectory(s.root); syncErr != nil {
			_ = os.Remove(artifact)
			return syncErr
		}
		if writeErr := atomicCreate(manifestPath, append(bytes, '\n'), 0o600); writeErr != nil {
			_ = os.Remove(artifact)
			_ = durablefs.SyncDirectory(s.root)
			return writeErr
		}
		return nil
	})
	if err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SetRollout changes how much of the fleet a published release reaches without
// re-uploading it. Staged rollout and an emergency stop are the two things an
// operator needs most, and both were previously impossible without republishing.
func (s *Store) SetRollout(base string, rollout int) (Manifest, error) {
	if rollout < 0 || rollout > 100 {
		return Manifest{}, errors.New("the rollout percent must be 0 to 100")
	}
	if !safeBase(base) {
		return Manifest{}, errors.New("invalid release identifier")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.root, base+".json")
	var manifest Manifest
	err := s.withFileLock(func() error {
		bytes, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		artifact := filepath.Join(s.root, base+".bin")
		if _, statErr := os.Stat(artifact); statErr != nil {
			// Never preserve or rewrite a manifest whose artifact vanished. This
			// also repairs a half-damaged shared store before returning the error.
			_ = os.Remove(path)
			_ = durablefs.SyncDirectory(s.root)
			return fmt.Errorf("release artifact is missing: %w", statErr)
		}
		if decodeErr := json.Unmarshal(bytes, &manifest); decodeErr != nil {
			return decodeErr
		}
		normalizeSignatureContract(&manifest)
		manifest.Rollout = rollout
		encoded, _ := json.MarshalIndent(manifest, "", "  ")
		return replaceFile(path, append(encoded, '\n'), 0o600)
	})
	if err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// Retire removes a release so no agent can be offered it again. The emergency
// action is SetRollout(0); this is the cleanup that follows.
func (s *Store) Retire(base string) error {
	if !safeBase(base) {
		return errors.New("invalid release identifier")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.withFileLock(func() error {
		if err := os.Remove(filepath.Join(s.root, base+".json")); err != nil {
			return err
		}
		_ = os.Remove(filepath.Join(s.root, base+".bin"))
		return durablefs.SyncDirectory(s.root)
	})
}

// Releases lists every published build, newest first.
func (s *Store) Releases() ([]Release, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	releases := make([]Release, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var manifest Manifest
		bytes, err := os.ReadFile(filepath.Join(s.root, entry.Name()))
		if err != nil || json.Unmarshal(bytes, &manifest) != nil {
			continue
		}
		normalizeSignatureContract(&manifest)
		releases = append(releases, Release{
			Manifest: manifest,
			Base:     strings.TrimSuffix(entry.Name(), ".json"),
		})
	}
	sort.Slice(releases, func(i, j int) bool {
		if difference := compareVersion(
			releases[i].Version, releases[j].Version,
		); difference != 0 {
			return difference > 0
		}
		return releases[i].Base < releases[j].Base
	})
	return releases, nil
}

func (s *Store) stageArtifact(
	base string,
	source io.Reader,
) (string, string, int64, error) {
	file, err := os.CreateTemp(s.root, "."+base+".staging-*")
	if err != nil {
		return "", "", 0, err
	}
	temporary := file.Name()
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		os.Remove(temporary)
		return "", "", 0, err
	}
	hash := sha256.New()
	size, err := io.Copy(
		io.MultiWriter(file, hash),
		io.LimitReader(source, 128*1024*1024+1),
	)
	if err != nil || size > 128*1024*1024 {
		file.Close()
		os.Remove(temporary)
		return "", "", 0, errors.New(
			"the artifact exceeds 128 MiB or could not be read",
		)
	}
	if size == 0 {
		file.Close()
		os.Remove(temporary)
		return "", "", 0, errors.New("the artifact is empty")
	}
	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(temporary)
		return "", "", 0, err
	}
	if err := file.Close(); err != nil {
		os.Remove(temporary)
		return "", "", 0, err
	}
	return temporary, hex.EncodeToString(hash.Sum(nil)), size, nil
}

func safeBase(value string) bool {
	return value != "" && !strings.Contains(value, "/") &&
		!strings.Contains(value, "..") && !strings.HasPrefix(value, ".")
}

func replaceFile(path string, bytes []byte, mode os.FileMode) error {
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
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
	return durablefs.SyncDirectory(filepath.Dir(path))
}

func (s *Store) Latest(channel, osName, architecture, agentID string) (*Manifest, error) {
	matches, err := s.Candidates(channel, osName, architecture, agentID)
	if err != nil || len(matches) == 0 {
		return nil, err
	}
	return &matches[0], nil
}

// Candidates returns rollout-eligible manifests newest first. The HTTP layer
// still has to apply the requesting Agent's protocol/version capability gate;
// returning the ordered set prevents a rollback that is unsafe for a legacy
// Agent from hiding a lower normal bridge upgrade.
func (s *Store) Candidates(
	channel, osName, architecture, agentID string,
) ([]Manifest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	var matches []Manifest
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var manifest Manifest
		bytes, err := os.ReadFile(filepath.Join(s.root, entry.Name()))
		if err != nil || json.Unmarshal(bytes, &manifest) != nil {
			continue
		}
		normalizeSignatureContract(&manifest)
		if manifest.Channel == channel && manifest.OS == osName &&
			manifest.Architecture == architecture &&
			manifest.Rollout > 0 &&
			rolloutBucket(agentID) < manifest.Rollout {
			matches = append(matches, manifest)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return compareVersion(matches[i].Version, matches[j].Version) > 0
	})
	return matches, nil
}

func (s *Store) Artifact(base string) (string, error) {
	if strings.Contains(base, "/") || strings.Contains(base, "..") {
		return "", errors.New("invalid update artifact")
	}
	path := filepath.Join(s.root, base+".bin")
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}

func rolloutBucket(agentID string) int {
	value := 0
	for _, char := range agentID {
		value = (value*31 + int(char)) % 100
	}
	return value
}

func compareVersion(left, right string) int {
	var l1, l2, l3, r1, r2, r3 int
	fmt.Sscanf(strings.TrimPrefix(left, "v"), "%d.%d.%d", &l1, &l2, &l3)
	fmt.Sscanf(strings.TrimPrefix(right, "v"), "%d.%d.%d", &r1, &r2, &r3)
	switch {
	case l1 != r1:
		return l1 - r1
	case l2 != r2:
		return l2 - r2
	default:
		return l3 - r3
	}
}

func safeVersion(value string) bool {
	var a, b, c int
	n, err := fmt.Sscanf(strings.TrimPrefix(value, "v"), "%d.%d.%d", &a, &b, &c)
	return err == nil && n == 3 && value == fmt.Sprintf("%d.%d.%d", a, b, c)
}

// atomicCreate writes a file that must not already exist.
//
// It links rather than renames, so it fails instead of replacing. That is
// deliberate here - a staged artifact and its manifest are signed, and silently
// overwriting one would lose the record of what was replaced. Callers that mean
// to replace remove the old path first.
//
// The bootstrap store has a function of the same shape called atomicWrite that
// renames, so it does replace. Two functions with one name and different
// answers to "what if it exists" is a trap; this one says which it is.
func atomicCreate(path string, bytes []byte, mode os.FileMode) error {
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
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
	if err := os.Link(temporary, path); err != nil {
		return err
	}
	return durablefs.SyncDirectory(filepath.Dir(path))
}
