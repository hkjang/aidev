package httpapi

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hkjang/invenqor/server/internal/mail"
	"github.com/hkjang/invenqor/server/internal/updates"
)

// Publishing a build used to be the only update operation the API offered, which
// left an operator with no way to widen a rollout, stop a bad release, or see
// how far one had progressed - short of republishing with a new version number.
// These endpoints exist so the normal rollout workflow needs no re-upload.

type releaseView struct {
	updates.Release
	// Adopted counts agents already reporting this version, and Eligible how
	// many the current rollout percent can reach. Rollout progress is the
	// question an operator asks immediately after publishing.
	Adopted  int64 `json:"adopted_agents"`
	Eligible int64 `json:"eligible_agents"`
}

func (s *Server) listAgentUpdates(
	response http.ResponseWriter,
	request *http.Request,
) {
	if s.updateStore == nil {
		writeAPIError(
			response, request, http.StatusServiceUnavailable,
			"UPDATE_STORE_UNAVAILABLE", "The update store is unavailable.",
		)
		return
	}
	releases, err := s.updateStore.Releases()
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	versions, total, err := s.agentVersionCounts(request.Context())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	views := make([]releaseView, 0, len(releases))
	for _, release := range releases {
		views = append(views, releaseView{
			Release:  release,
			Adopted:  versions[release.Version],
			Eligible: total * int64(release.Rollout) / 100,
		})
	}
	distribution := make([]statisticBucket, 0, len(versions))
	for version, count := range versions {
		label := version
		if label == "" {
			label = "미확인"
		}
		distribution = append(distribution, statisticBucket{
			Label: label, Count: count,
		})
	}
	sortBucketsByCount(distribution)
	writeJSON(response, http.StatusOK, map[string]any{
		"releases":            views,
		"agent_versions":      distribution,
		"agents":              total,
		"signature_verified":  s.updateStore.SigningKeyConfigured(),
		"agent_version":       "",
		"signing_key_missing": !s.updateStore.SigningKeyConfigured(),
	})
}

func (s *Server) agentVersionCounts(
	ctx context.Context,
) (map[string]int64, int64, error) {
	rows, err := s.database.DB().QueryContext(
		ctx,
		`SELECT COALESCE(NULLIF(version,''),'') AS version, COUNT(*)
		   FROM agents WHERE blocked_at IS NULL
		  GROUP BY version`,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	counts := map[string]int64{}
	var total int64
	for rows.Next() {
		var version string
		var count int64
		if err := rows.Scan(&version, &count); err != nil {
			return nil, 0, err
		}
		counts[version] = count
		total += count
	}
	return counts, total, rows.Err()
}

func sortBucketsByCount(buckets []statisticBucket) {
	for outer := 1; outer < len(buckets); outer++ {
		for index := outer; index > 0; index-- {
			left, right := buckets[index-1], buckets[index]
			if right.Count < left.Count ||
				(right.Count == left.Count && right.Label >= left.Label) {
				break
			}
			buckets[index-1], buckets[index] = right, left
		}
	}
}

func (s *Server) updateAgentUpdateRollout(
	response http.ResponseWriter,
	request *http.Request,
) {
	if s.updateStore == nil {
		writeAPIError(
			response, request, http.StatusServiceUnavailable,
			"UPDATE_STORE_UNAVAILABLE", "The update store is unavailable.",
		)
		return
	}
	base := chi.URLParam(request, "release")
	var input struct {
		RolloutPercent *int   `json:"rollout_percent"`
		Reason         string `json:"reason"`
	}
	if base == "" || decodeJSON(request, &input) != nil || input.RolloutPercent == nil {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_REQUEST", "rollout_percent is required.",
		)
		return
	}
	manifest, err := s.updateStore.SetRollout(base, *input.RolloutPercent)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeAPIError(
				response, request, http.StatusNotFound,
				"RELEASE_NOT_FOUND", "The release does not exist.",
			)
			return
		}
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_ROLLOUT", err.Error(),
		)
		return
	}
	action := "agent_update.rollout"
	if *input.RolloutPercent == 0 {
		// Stopping a release is the action an operator takes under pressure, so it
		// gets its own audit verb.
		action = "agent_update.halt"
	}
	s.recordAdminAudit(
		request, action, "agent_update", base, nil,
		map[string]any{
			"version":         manifest.Version,
			"architecture":    manifest.Architecture,
			"rollout_percent": manifest.Rollout,
		},
		input.Reason,
	)
	if *input.RolloutPercent == 0 {
		s.notifyMail(
			request.Context(),
			mail.AgentUpdateHalted(
				manifest.Version, manifest.Architecture, actorLabel(request),
				input.Reason,
			),
			principalFromContext(request.Context()).User.ID,
			s.superAdminIDs(request.Context()),
		)
	}
	writeJSON(response, http.StatusOK, map[string]any{"release": manifest})
}

func (s *Server) retireAgentUpdate(
	response http.ResponseWriter,
	request *http.Request,
) {
	if s.updateStore == nil {
		writeAPIError(
			response, request, http.StatusServiceUnavailable,
			"UPDATE_STORE_UNAVAILABLE", "The update store is unavailable.",
		)
		return
	}
	base := chi.URLParam(request, "release")
	if err := s.updateStore.Retire(base); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeAPIError(
				response, request, http.StatusNotFound,
				"RELEASE_NOT_FOUND", "The release does not exist.",
			)
			return
		}
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_RELEASE", err.Error(),
		)
		return
	}
	s.recordAdminAudit(
		request, "agent_update.retire", "agent_update", base, nil,
		map[string]any{"retired": true}, "",
	)
	response.WriteHeader(http.StatusNoContent)
}

type agentUpdatePublishOptions struct {
	Version, Channel, OS, Architecture string
	SHA256                             string
	Size                               int64
	Signature, ManifestSignature       string
	AllowDowngrade                     bool
	Notes                              string
	Rollout                            int
}

// updateSignatureBundle is exactly the JSON emitted by the offline signing
// helper. Uploading this one small file avoids copying two signatures and seven
// identity fields by hand while still detecting a form/bundle mismatch.
type updateSignatureBundle struct {
	Version           string `json:"version"`
	Channel           string `json:"channel"`
	OS                string `json:"os"`
	Architecture      string `json:"architecture"`
	Size              int64  `json:"size"`
	SHA256            string `json:"sha256"`
	AllowDowngrade    bool   `json:"allow_downgrade"`
	SignatureScheme   string `json:"signature_scheme"`
	SignatureVersion  int    `json:"signature_version"`
	Signature         string `json:"signature"`
	ManifestSignature string `json:"manifest_signature"`
}

func uploadedSignature(request *http.Request, field, fileField string) (string, error) {
	value := strings.TrimSpace(request.FormValue(field))
	file, _, err := request.FormFile(fileField)
	if errors.Is(err, http.ErrMissingFile) {
		return value, nil
	}
	if err != nil {
		return "", fmt.Errorf("the %s file could not be opened", field)
	}
	defer file.Close()
	contents, err := io.ReadAll(io.LimitReader(file, 4097))
	if err != nil || len(contents) > 4096 {
		return "", fmt.Errorf("the %s file could not be read or exceeds 4096 bytes", field)
	}
	// openssl pkeyutl writes 64 raw bytes. Accept that output directly as well
	// as a base64 text file.
	fileValue := strings.TrimSpace(string(contents))
	if len(contents) == ed25519.SignatureSize {
		fileValue = base64.StdEncoding.EncodeToString(contents)
	}
	if value != "" {
		direct, directErr := updates.DecodeSignature(value)
		uploaded, uploadedErr := updates.DecodeSignature(fileValue)
		if directErr != nil || uploadedErr != nil || !bytes.Equal(direct, uploaded) {
			return "", fmt.Errorf("%s does not match the uploaded %s", field, fileField)
		}
	}
	return fileValue, nil
}

func uploadedSignatureBundle(request *http.Request) (*updateSignatureBundle, error) {
	file, _, err := request.FormFile("signature_bundle_file")
	if errors.Is(err, http.ErrMissingFile) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.New("the signature bundle could not be opened")
	}
	defer file.Close()
	contents, err := io.ReadAll(io.LimitReader(file, 16*1024+1))
	if err != nil || len(contents) > 16*1024 {
		return nil, errors.New("the signature bundle could not be read or exceeds 16384 bytes")
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var bundle updateSignatureBundle
	if err := decoder.Decode(&bundle); err != nil {
		return nil, fmt.Errorf("decode signature bundle: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("the signature bundle contains trailing JSON data")
	}
	if bundle.SignatureScheme != updates.SignatureSchemeEd25519 ||
		bundle.SignatureVersion != updates.SignatureVersionV2 {
		return nil, errors.New("the signature bundle must use ed25519 manifest signature version 2")
	}
	if bundle.Size <= 0 || len(bundle.SHA256) != 64 ||
		bundle.SHA256 != strings.ToLower(bundle.SHA256) {
		return nil, errors.New("the signature bundle size or sha256 is invalid")
	}
	if _, err := hex.DecodeString(bundle.SHA256); err != nil {
		return nil, errors.New("the signature bundle sha256 is invalid")
	}
	return &bundle, nil
}

func mergeBundleString(field, formValue, bundleValue string) (string, error) {
	formValue = strings.TrimSpace(formValue)
	if formValue != "" && formValue != bundleValue {
		return "", fmt.Errorf("%s does not match the signature bundle", field)
	}
	if formValue != "" {
		return formValue, nil
	}
	return bundleValue, nil
}

func mergeBundleSignature(field, direct, bundled string) (string, error) {
	if direct == "" {
		return bundled, nil
	}
	if bundled == "" {
		return direct, nil
	}
	left, leftErr := updates.DecodeSignature(direct)
	right, rightErr := updates.DecodeSignature(bundled)
	if leftErr != nil || rightErr != nil || !bytes.Equal(left, right) {
		return "", fmt.Errorf("%s does not match the signature bundle", field)
	}
	return direct, nil
}

func optionalFormBool(request *http.Request, field string) (bool, bool, error) {
	values, present := request.MultipartForm.Value[field]
	if !present {
		return false, false, nil
	}
	if len(values) != 1 {
		return false, true, fmt.Errorf("%s must be provided once", field)
	}
	switch strings.ToLower(strings.TrimSpace(values[0])) {
	case "true":
		return true, true, nil
	case "false":
		return false, true, nil
	default:
		return false, true, fmt.Errorf("%s must be true or false", field)
	}
}

// publishOptions accepts either the helper's single JSON bundle or two direct
// Base64/raw signature fields. If both are present they must describe exactly
// the same release; no source silently overrides another.
func publishOptions(request *http.Request) (agentUpdatePublishOptions, error) {
	var options agentUpdatePublishOptions
	bundle, err := uploadedSignatureBundle(request)
	if err != nil {
		return options, err
	}
	legacy, err := uploadedSignature(request, "signature", "signature_file")
	if err != nil {
		return options, err
	}
	manifest, err := uploadedSignature(
		request, "manifest_signature", "manifest_signature_file",
	)
	if err != nil {
		return options, err
	}
	formAllowDowngrade, allowDowngradePresent, err := optionalFormBool(
		request, "allow_downgrade",
	)
	if err != nil {
		return options, err
	}

	if bundle != nil {
		if options.Version, err = mergeBundleString(
			"version", request.FormValue("version"), bundle.Version,
		); err != nil {
			return options, err
		}
		if options.Channel, err = mergeBundleString(
			"channel", request.FormValue("channel"), bundle.Channel,
		); err != nil {
			return options, err
		}
		if options.OS, err = mergeBundleString(
			"os", request.FormValue("os"), bundle.OS,
		); err != nil {
			return options, err
		}
		if options.Architecture, err = mergeBundleString(
			"architecture", request.FormValue("architecture"), bundle.Architecture,
		); err != nil {
			return options, err
		}
		if options.Signature, err = mergeBundleSignature(
			"signature", legacy, bundle.Signature,
		); err != nil {
			return options, err
		}
		if options.ManifestSignature, err = mergeBundleSignature(
			"manifest_signature", manifest, bundle.ManifestSignature,
		); err != nil {
			return options, err
		}
		options.Size, options.SHA256 = bundle.Size, bundle.SHA256
		options.AllowDowngrade = bundle.AllowDowngrade
		if allowDowngradePresent {
			if formAllowDowngrade != bundle.AllowDowngrade {
				return options, errors.New("allow_downgrade does not match the signature bundle")
			}
		}
	} else {
		options.Version = strings.TrimSpace(request.FormValue("version"))
		options.Channel = strings.TrimSpace(request.FormValue("channel"))
		options.OS = strings.TrimSpace(request.FormValue("os"))
		options.Architecture = strings.TrimSpace(request.FormValue("architecture"))
		options.Signature = legacy
		options.ManifestSignature = manifest
		options.AllowDowngrade = formAllowDowngrade
	}
	if options.Channel == "" {
		options.Channel = "stable"
	}
	if options.OS == "" {
		options.OS = "linux"
	}
	if options.Signature == "" || options.ManifestSignature == "" {
		return options, errors.New("signature and manifest_signature are both required")
	}
	options.Rollout = 100
	if raw := strings.TrimSpace(request.FormValue("rollout_percent")); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil {
			return options, errors.New("rollout_percent must be a number")
		}
		options.Rollout = parsed
	}
	options.Notes = strings.TrimSpace(request.FormValue("notes"))
	return options, nil
}
