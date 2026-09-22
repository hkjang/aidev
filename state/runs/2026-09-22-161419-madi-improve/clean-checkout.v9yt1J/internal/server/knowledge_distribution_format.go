package server

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"path"
	"strings"
	"time"
)

const distributionMaxBytes = 50 << 20
const distributionMaxFiles = 1000
const distributionMaxManifest = 2 << 20
const distributionManifestPath = "madi-distribution.json"
const distributionSignaturePath = "madi-distribution.sig"

type distributionMetadata struct {
	Title   string   `json:"title"`
	Tags    []string `json:"tags"`
	Aliases []string `json:"aliases"`
	Icon    string   `json:"icon"`
}

func (m distributionMetadata) value() map[string]any {
	return map[string]any{"title": m.Title, "tags": m.Tags, "aliases": m.Aliases, "icon": m.Icon}
}

type distributionApproval struct {
	Required     bool   `json:"required"`
	RequestID    string `json:"request_id"`
	ResourceHash string `json:"resource_hash"`
}
type distributionFile struct {
	SourceID       string               `json:"source_id"`
	Path           string               `json:"path"`
	Kind           string               `json:"kind"`
	Bytes          int64                `json:"bytes"`
	SHA256         string               `json:"sha256"`
	ParentSourceID string               `json:"parent_source_id"`
	SourceVersion  int                  `json:"source_version"`
	Metadata       distributionMetadata `json:"metadata"`
	MetadataHash   string               `json:"metadata_hash"`
	Approval       distributionApproval `json:"approval"`
}
type distributionManifest struct {
	Format             string                `json:"format"`
	BundleID           string                `json:"bundle_id"`
	SourceInstance     string                `json:"source_instance"`
	SourceWorkspace    string                `json:"source_workspace"`
	ReceiverInstance   string                `json:"receiver_instance"`
	KeyID              string                `json:"key_id"`
	ProtectionRevision int                   `json:"protection_revision"`
	CreatedAt          int64                 `json:"created_at"`
	ExpiresAt          int64                 `json:"expires_at"`
	Files              []distributionFile    `json:"files"`
	BundleApproval     *distributionApproval `json:"bundle_approval,omitempty"`
}

func (m distributionManifest) sourceKey() string {
	return "signed:" + m.SourceInstance + ":" + m.SourceWorkspace
}

// madi-distribution-v1 signs this exact Go JSON representation, not arbitrary
// JSON or RFC 8785. Re-encoding equality rejects duplicate/unknown keys, numbers
// in alternative encodings, extra whitespace and ambiguous signed structures.
func parseDistributionManifest(raw []byte) (distributionManifest, error) {
	var m distributionManifest
	invalid := errors.New("서명 매니페스트 형식·경로·해시·크기·참조를 확인하세요")
	if len(raw) == 0 || len(raw) > distributionMaxManifest || json.Unmarshal(raw, &m) != nil || !bytes.Equal(raw, jsonValue(m)) {
		return m, invalid
	}
	if m.Format != "madi-distribution-v1" || !validID(m.BundleID) || !validID(m.SourceInstance) || !validID(m.SourceWorkspace) || !validID(m.ReceiverInstance) || !validID(m.KeyID) || m.ProtectionRevision < 1 || m.ProtectionRevision > 2147483647 || m.CreatedAt < 1 || m.ExpiresAt <= m.CreatedAt || m.ExpiresAt-m.CreatedAt > 365*86400 || len(m.Files) == 0 || len(m.Files) > distributionMaxFiles {
		return m, invalid
	}
	if a := m.BundleApproval; a != nil && (!a.Required || !validID(a.RequestID) || !migrationHexHash(a.ResourceHash)) {
		return m, invalid
	}
	ids := map[string]distributionFile{}
	names := map[string]bool{}
	var total int64
	for _, f := range m.Files {
		if !validID(f.SourceID) || !oneOf(f.Kind, "document", "attachment") || !safeVaultPath(f.Path) || len(f.Path) > 512 || strings.Count(f.Path, "/") >= 20 || strings.HasPrefix(path.Base(f.Path), ".madi-") || oneOf(strings.Split(f.Path, "/")[0], ".git", ".obsidian", "__MACOSX") || oneOf(f.Path, distributionManifestPath, distributionSignaturePath) || strings.HasSuffix(f.Path, "/") || f.Bytes < 0 || f.Bytes > distributionMaxBytes || f.Kind == "document" && f.Bytes > 4<<20 || !migrationHexHash(f.SHA256) || names[f.Path] || ids[f.SourceID].SourceID != "" {
			return m, invalid
		}
		if f.MetadataHash != digest(string(jsonValue(f.Metadata.value()))) || len(jsonValue(f.Metadata)) > 64<<10 || f.Metadata.Tags == nil || f.Metadata.Aliases == nil {
			return m, invalid
		}
		if f.Kind == "document" && (f.SourceVersion < 1 || f.SourceVersion > 2147483647) || f.Kind == "attachment" && f.SourceVersion != 0 {
			return m, invalid
		}
		if f.Approval.Required {
			if f.Kind != "document" || !validID(f.Approval.RequestID) || !migrationHexHash(f.Approval.ResourceHash) {
				return m, invalid
			}
		} else if f.Approval.RequestID != "" || f.Approval.ResourceHash != "" {
			return m, invalid
		}
		total += f.Bytes
		if total > distributionMaxBytes {
			return m, invalid
		}
		names[f.Path] = true
		ids[f.SourceID] = f
	}
	for _, f := range m.Files {
		if f.Kind == "attachment" && f.ParentSourceID == "" {
			return m, invalid
		}
		seen := map[string]bool{f.SourceID: true}
		parent := f.ParentSourceID
		for parent != "" {
			v, ok := ids[parent]
			if !ok || v.Kind != "document" || seen[parent] || len(seen) >= 20 {
				return m, invalid
			}
			seen[parent] = true
			parent = v.ParentSourceID
		}
	}
	return m, nil
}
func verifyDistributionSignature(raw []byte, signature, public string) error {
	key, e := base64.StdEncoding.Strict().DecodeString(public)
	if e != nil || len(key) != ed25519.PublicKeySize {
		return errors.New("등록된 공개키 형식이 올바르지 않습니다")
	}
	sig, e := base64.StdEncoding.Strict().DecodeString(signature)
	if e != nil || len(sig) != ed25519.SignatureSize || !ed25519.Verify(key, raw, sig) {
		return errors.New("서명 검증에 실패했습니다. 반출망 관리자와 공개키 지문을 별도로 확인하세요")
	}
	return nil
}
func distributionCurrentTime(m distributionManifest, now time.Time, maxDays int) error {
	if m.CreatedAt > now.Unix()+300 || m.ExpiresAt <= now.Unix() || m.ExpiresAt-m.CreatedAt > int64(maxDays)*86400 {
		return errors.New("배포 패키지가 만료됐거나 현재 유효기간·시계 정책을 만족하지 않습니다")
	}
	return nil
}

// ZIP member names are never materialized as filesystem paths. Verify every
// uncompressed byte, CRC and declared SHA-256, without trusting ZIP size fields.
func readDistributionArchive(ctx context.Context, raw []byte) (distributionManifest, []byte, string, map[string][]byte, error) {
	var empty distributionManifest
	invalid := errors.New("서명 ZIP의 파일 목록·크기·내용 해시가 일치하지 않습니다")
	if len(raw) > distributionMaxBytes+distributionMaxManifest+(2<<20) {
		return empty, nil, "", nil, invalid
	}
	z, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if e != nil || len(z.File) > distributionMaxFiles+2 {
		return empty, nil, "", nil, invalid
	}
	files := map[string][]byte{}
	var total int64
	for _, f := range z.File {
		_, exists := files[f.Name]
		if !safeVaultPath(f.Name) || !f.Mode().IsRegular() || f.UncompressedSize64 > distributionMaxBytes || exists {
			return empty, nil, "", nil, invalid
		}
		limit := int64(distributionMaxBytes)
		if f.Name == distributionManifestPath {
			limit = distributionMaxManifest
		}
		if f.Name == distributionSignaturePath {
			limit = 128
		}
		reader, err := f.Open()
		if err != nil {
			return empty, nil, "", nil, invalid
		}
		data, err := io.ReadAll(io.LimitReader(contextVaultReader{ctx, reader}, limit+1))
		reader.Close()
		total += int64(len(data))
		if err != nil || int64(len(data)) > limit || total > distributionMaxBytes+distributionMaxManifest+128 {
			return empty, nil, "", nil, invalid
		}
		files[f.Name] = data
	}
	rawManifest := files[distributionManifestPath]
	signature := string(files[distributionSignaturePath])
	delete(files, distributionManifestPath)
	delete(files, distributionSignaturePath)
	m, e := parseDistributionManifest(rawManifest)
	if e != nil {
		return m, nil, "", nil, e
	}
	if len(files) != len(m.Files) {
		return m, nil, "", nil, invalid
	}
	for _, f := range m.Files {
		data, ok := files[f.Path]
		if !ok || int64(len(data)) != f.Bytes || digest(string(data)) != f.SHA256 {
			return m, nil, "", nil, invalid
		}
	}
	return m, rawManifest, signature, files, nil
}
