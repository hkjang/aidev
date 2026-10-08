package spool

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hkjang/invenqor/server/internal/durablefs"
)

type Manager struct {
	directory string
}

type ReplayLease struct {
	file *os.File
}

const (
	stagingPrefix       = ".incoming-event-"
	stagingCleanupGrace = time.Hour
)

func Open(stateDir string) (*Manager, error) {
	return OpenDirectory(filepath.Join(stateDir, "spool"))
}

// OpenDirectory opens an explicit spool directory. Kubernetes uses this with
// an RWX volume shared by every Server Pod; Open retains the Pod-local default
// used by single-node and compose installations.
func OpenDirectory(directory string) (*Manager, error) {
	if !filepath.IsAbs(directory) {
		return nil, errors.New("event spool configuration is invalid")
	}
	if err := durablefs.EnsurePrivateDirectory(directory); err != nil {
		return nil, fmt.Errorf("secure event spool: %w", err)
	}
	if err := cleanupStagingSegments(directory, time.Now()); err != nil {
		return nil, fmt.Errorf("clean event spool staging files: %w", err)
	}
	return &Manager{directory: directory}, nil
}

// Append creates an immutable event segment. Existing names are treated as
// idempotent duplicate writes and are never overwritten.
func (m *Manager) Append(
	agentID string,
	eventID string,
	raw []byte,
) (bool, error) {
	if !safeID(agentID) || !safeID(eventID) {
		return false, errors.New("spool identity is invalid")
	}
	finalPath := filepath.Join(m.directory, agentID+"_"+eventID+".json")
	file, err := os.CreateTemp(m.directory, stagingPrefix+"*.tmp")
	if err != nil {
		return false, fmt.Errorf("create spool staging segment: %w", err)
	}
	stagingPath := file.Name()
	defer func() {
		_ = file.Close()
		_ = os.Remove(stagingPath)
	}()
	locked, err := tryLockReplayFile(file)
	if err != nil {
		return false, fmt.Errorf("lock spool staging segment: %w", err)
	}
	if !locked {
		return false, errors.New("new spool staging segment could not be locked")
	}
	if err := file.Chmod(0o600); err != nil {
		return false, fmt.Errorf("secure spool staging segment: %w", err)
	}
	if _, err := file.Write(raw); err != nil {
		return false, fmt.Errorf("write spool staging segment: %w", err)
	}
	if err := file.Sync(); err != nil {
		return false, fmt.Errorf("sync spool staging segment: %w", err)
	}
	if err := file.Close(); err != nil {
		return false, fmt.Errorf("close spool staging segment: %w", err)
	}
	// Hard-link publication is an atomic no-clobber operation on the shared
	// filesystem: a replayer sees either no final segment or the fully synced
	// inode. Concurrent retries for the same event cannot overwrite the winner.
	if err := os.Link(stagingPath, finalPath); errors.Is(err, fs.ErrExist) {
		if removeErr := os.Remove(stagingPath); removeErr != nil &&
			!errors.Is(removeErr, fs.ErrNotExist) {
			return false, fmt.Errorf("remove duplicate spool staging segment: %w", removeErr)
		}
		if err := durablefs.SyncDirectory(m.directory); err != nil {
			return false, fmt.Errorf("sync duplicate spool directory: %w", err)
		}
		return true, nil
	} else if err != nil {
		return false, fmt.Errorf("publish spool segment: %w", err)
	}
	if err := os.Remove(stagingPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("remove published spool staging segment: %w", err)
	}
	if err := durablefs.SyncDirectory(m.directory); err != nil {
		return false, fmt.Errorf("sync spool directory: %w", err)
	}
	return false, nil
}

func cleanupStagingSegments(directory string, now time.Time) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	removed := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), stagingPrefix) ||
			!strings.HasSuffix(entry.Name(), ".tmp") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		// Do not race a writer from another Pod that has only just created its
		// stage. A crashed stage is never replay-visible and is reclaimed on a
		// later open after this conservative grace period.
		if info.ModTime().After(now.Add(-stagingCleanupGrace)) {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		candidate, err := os.OpenFile(path, os.O_RDWR, 0)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		locked, lockErr := tryLockReplayFile(candidate)
		if lockErr != nil {
			_ = candidate.Close()
			return lockErr
		}
		if !locked {
			_ = candidate.Close()
			continue
		}
		_ = unlockReplayFile(candidate)
		if err := candidate.Close(); err != nil {
			return err
		}
		if err := os.Remove(path); err != nil &&
			!errors.Is(err, fs.ErrNotExist) {
			return err
		}
		removed = true
	}
	if removed {
		return durablefs.SyncDirectory(directory)
	}
	return nil
}

// AcquireReplayLease elects one replayer across all Pods sharing the RWX
// directory. The persistent file is protected by an OS advisory lock, which
// the kernel releases if the owning process crashes.
func (m *Manager) AcquireReplayLease() (*ReplayLease, bool, error) {
	path := filepath.Join(m.directory, ".replay.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false, fmt.Errorf("open replay lock: %w", err)
	}
	acquired, err := tryLockReplayFile(file)
	if err != nil {
		_ = file.Close()
		return nil, false, fmt.Errorf("lock replay spool: %w", err)
	}
	if !acquired {
		_ = file.Close()
		return nil, false, nil
	}
	if err := cleanupStagingSegments(m.directory, time.Now()); err != nil {
		_ = unlockReplayFile(file)
		_ = file.Close()
		return nil, false, fmt.Errorf("clean crashed spool staging files: %w", err)
	}
	return &ReplayLease{file: file}, true, nil
}

func (lease *ReplayLease) Renew() error {
	if lease == nil || lease.file == nil {
		return errors.New("replay lease is invalid")
	}
	return nil
}

func (lease *ReplayLease) Release() error {
	if lease == nil || lease.file == nil {
		return nil
	}
	err := unlockReplayFile(lease.file)
	closeErr := lease.file.Close()
	lease.file = nil
	if err != nil {
		return fmt.Errorf("unlock replay spool: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("close replay lock: %w", closeErr)
	}
	return nil
}

func (m *Manager) Pending() ([]string, error) {
	entries, err := os.ReadDir(m.directory)
	if err != nil {
		return nil, err
	}
	type segment struct {
		path      string
		arrivedAt time.Time
		eventID   string
	}
	segments := make([]segment, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			path := filepath.Join(m.directory, entry.Name())
			item := segment{path: path}
			if info, statErr := entry.Info(); statErr == nil {
				item.arrivedAt = info.ModTime()
			}
			// Older releases named files from UUIDs, so directory/lexical order is
			// unrelated to durable server arrival. Use filesystem mtime as arrival
			// order and read event_id only as a stable tie-breaker; the replay layer
			// remains responsible for full validation and reporting.
			var metadata struct {
				CreatedAt uint64 `json:"created_at"`
				EventID   string `json:"event_id"`
			}
			if raw, readErr := os.ReadFile(path); readErr == nil &&
				json.Unmarshal(raw, &metadata) == nil &&
				metadata.CreatedAt > 0 && metadata.EventID != "" {
				item.eventID = metadata.EventID
			}
			segments = append(segments, item)
		}
	}
	sort.Slice(segments, func(left, right int) bool {
		first, second := segments[left], segments[right]
		if !first.arrivedAt.Equal(second.arrivedAt) {
			return first.arrivedAt.Before(second.arrivedAt)
		}
		if first.eventID != second.eventID {
			return first.eventID < second.eventID
		}
		return first.path < second.path
	})
	result := make([]string, len(segments))
	for index := range segments {
		result[index] = segments[index].path
	}
	return result, nil
}

func (m *Manager) Read(path string) ([]byte, error) {
	if !m.ownsPath(path) {
		return nil, errors.New("spool path is outside the spool directory")
	}
	return os.ReadFile(path)
}

func (m *Manager) Acknowledge(path string) error {
	if !m.ownsPath(path) {
		return errors.New("spool path is outside the spool directory")
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("acknowledge spool segment: %w", err)
	}
	if err := durablefs.SyncDirectory(m.directory); err != nil {
		return fmt.Errorf("sync acknowledged spool segment: %w", err)
	}
	return nil
}

func (m *Manager) Directory() string {
	return m.directory
}

func (m *Manager) ownsPath(path string) bool {
	return filepath.Dir(path) == m.directory && filepath.Base(path) != "."
}

func safeID(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' {
			continue
		}
		return false
	}
	return true
}
