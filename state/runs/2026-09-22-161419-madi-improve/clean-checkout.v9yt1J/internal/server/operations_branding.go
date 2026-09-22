package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var brandDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)
var brandColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (s *Server) registerBrandOperations() {
	s.handle("POST /api/v1/workspaces/{id}/branding/assets", s.uploadBrandAsset)
	s.handle("GET /api/v1/workspaces/{id}/branding/assets", s.listBrandAssets)
	s.handle("GET /api/v1/workspaces/{id}/branding/assets/{digest}", s.getBrandAsset)
}
func brandAssetURL(wid, digest string) string {
	return "/api/v1/workspaces/" + wid + "/branding/assets/" + digest
}

// Raster-only, decoded under pixel and compressed-size budgets, then re-encoded
// without EXIF, profiles, comments, SVG URLs or other source metadata.
func sanitizeBrandImage(source []byte) ([]byte, int, int, error) {
	if len(source) == 0 || len(source) > 1<<20 {
		return nil, 0, 0, errors.New("브랜딩 이미지는 1MB 이하 PNG 또는 JPEG 파일이어야 합니다")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(source))
	if err != nil || !oneOf(format, "png", "jpeg") || config.Width < 1 || config.Height < 1 || config.Width > 2048 || config.Height > 2048 {
		return nil, 0, 0, errors.New("PNG/JPEG 형식과 최대 2048×2048 픽셀 제한을 확인하세요")
	}
	original, _, err := image.Decode(bytes.NewReader(source))
	if err != nil {
		return nil, 0, 0, errors.New("이미지를 안전하게 읽지 못했습니다")
	}
	width, height := config.Width, config.Height
	if width > 512 || height > 512 {
		scale := math.Min(512/float64(width), 512/float64(height))
		width = max(1, int(math.Round(float64(width)*scale)))
		height = max(1, int(math.Round(float64(height)*scale)))
	}
	clean := image.NewNRGBA(image.Rect(0, 0, width, height))
	bounds := original.Bounds()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			clean.Set(x, y, original.At(bounds.Min.X+x*config.Width/width, bounds.Min.Y+y*config.Height/height))
		}
	}
	var output bytes.Buffer
	if err = png.Encode(&output, clean); err != nil || output.Len() > 512<<10 {
		return nil, 0, 0, errors.New("변환 후 PNG가 512KB를 초과합니다. 더 단순하거나 작은 이미지를 사용하세요")
	}
	return output.Bytes(), width, height, nil
}
func (s *Server) uploadBrandAsset(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "브랜딩 이미지는 워크스페이스 관리자만 등록할 수 있습니다")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, (1<<20)+(64<<10))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		apiError(w, 400, "1MB 이하 PNG/JPEG 파일을 선택하세요")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		apiError(w, 400, "업로드할 파일을 선택하세요")
		return
	}
	defer file.Close()
	source, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil {
		apiError(w, 400, "이미지를 읽지 못했습니다")
		return
	}
	clean, width, height, err := sanitizeBrandImage(source)
	if err != nil {
		apiError(w, 400, err.Error())
		return
	}
	hash := sha256.Sum256(clean)
	digest := hex.EncodeToString(hash[:])
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	if _, err = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1,56102))", wid); err != nil {
		respond(w, nil, err)
		return
	}
	var count int
	var exists bool
	if err = tx.QueryRow(r.Context(), "SELECT count(*),coalesce(bool_or(digest=$2),false) FROM workspace_brand_assets WHERE workspace_id=$1", wid, digest).Scan(&count, &exists); err != nil {
		respond(w, nil, err)
		return
	}
	if count >= 40 && !exists {
		apiError(w, 400, "워크스페이스당 브랜딩 이미지 저장 한도는 40개입니다. 기존 이미지를 다시 선택하세요")
		return
	}
	var storedDigest string
	err = tx.QueryRow(r.Context(), `INSERT INTO workspace_brand_assets(workspace_id,digest,data,width,height,created_by) SELECT $1,$2,$3,$4,$5,$6 WHERE EXISTS(SELECT 1 FROM users u JOIN workspace_members m ON m.user_id=u.id WHERE u.id=$6 AND NOT u.disabled AND u.role<>'viewer' AND m.workspace_id=$1 AND m.role IN ('owner','admin')) ON CONFLICT(workspace_id,digest) DO UPDATE SET digest=EXCLUDED.digest RETURNING digest`, wid, digest, clean, width, height, current(r).ID).Scan(&storedDigest)
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "브랜딩 관리 권한이 변경되었습니다")
		return
	}
	s.audit(r, "WORKSPACE_BRAND_UPLOAD", wid, map[string]any{"digest": digest, "width": width, "height": height, "size_bytes": len(clean)})
	jsonResponse(w, 200, map[string]any{"url": brandAssetURL(wid, digest), "digest": digest, "width": width, "height": height, "size_bytes": len(clean), "notice": "외부 연결·메타데이터 없는 PNG로 변환했습니다. 설정을 저장하면 구성원에게 적용됩니다."})
}
func (s *Server) listBrandAssets(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "브랜딩 관리 권한이 없습니다")
		return
	}
	v, err := s.rows(r.Context(), `SELECT jsonb_build_object('digest',digest,'width',width,'height',height,'size_bytes',octet_length(data),'created_at',created_at) FROM workspace_brand_assets WHERE workspace_id=$1 AND EXISTS(SELECT 1 FROM users u JOIN workspace_members m ON m.user_id=u.id WHERE u.id=$2 AND NOT u.disabled AND u.role<>'viewer' AND m.workspace_id=$1 AND m.role IN ('owner','admin')) ORDER BY created_at DESC LIMIT 40`, wid, current(r).ID)
	for _, asset := range v {
		asset["url"] = brandAssetURL(wid, str(asset, "digest"))
	}
	respond(w, v, err)
}
func (s *Server) getBrandAsset(w http.ResponseWriter, r *http.Request) {
	wid, digest := r.PathValue("id"), r.PathValue("digest")
	if !validID(wid) || !brandDigest.MatchString(digest) || !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 404, "브랜딩 이미지를 찾을 수 없습니다")
		return
	}
	var data []byte
	err := s.DB.QueryRow(r.Context(), `SELECT a.data FROM workspace_brand_assets a JOIN workspace_members m ON m.workspace_id=a.workspace_id AND m.user_id=$3 JOIN users u ON u.id=m.user_id AND NOT u.disabled WHERE a.workspace_id=$1 AND a.digest=$2`, wid, digest, current(r).ID).Scan(&data)
	if err != nil {
		apiError(w, 404, "브랜딩 이미지를 찾을 수 없습니다")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.Header().Set("Content-Disposition", "inline; filename=branding.png")
	w.Write(data)
}
func whiteContrast(color string) float64 {
	if !brandColor.MatchString(color) {
		return 0
	}
	n, _ := strconv.ParseUint(strings.TrimPrefix(color, "#"), 16, 32)
	linear := func(component uint64) float64 {
		value := float64(component) / 255
		if value <= 0.04045 {
			return value / 12.92
		}
		return math.Pow((value+0.055)/1.055, 2.4)
	}
	luminance := 0.2126*linear((n>>16)&255) + 0.7152*linear((n>>8)&255) + 0.0722*linear(n&255)
	return 1.05 / (luminance + 0.05)
}
func validateWorkspaceBranding(ctx context.Context, q collaborationQuery, wid string, data map[string]any) error {
	for _, key := range []string{"logo_url", "favicon_url"} {
		if raw, exists := data[key]; exists {
			path, ok := raw.(string)
			if !ok {
				return errors.New("브랜딩 이미지 경로 형식을 확인하세요")
			}
			if path == "" || path == "/favicon.svg" {
				continue
			}
			prefix := "/api/v1/workspaces/" + wid + "/branding/assets/"
			if !strings.HasPrefix(path, prefix) || !brandDigest.MatchString(strings.TrimPrefix(path, prefix)) {
				return errors.New("이 워크스페이스에서 안전하게 업로드한 브랜딩 이미지만 선택할 수 있습니다")
			}
			var found bool
			if err := q.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM workspace_brand_assets WHERE workspace_id=$1 AND digest=$2)", wid, strings.TrimPrefix(path, prefix)).Scan(&found); err != nil || !found {
				return errors.New("선택한 브랜딩 이미지를 찾을 수 없습니다")
			}
		}
	}
	if raw, exists := data["theme_primary"]; exists {
		color, ok := raw.(string)
		if !ok || !brandColor.MatchString(color) || whiteContrast(color) < 4.5 {
			return errors.New("테마 색상은 #RRGGBB 형식이며 흰색 글자와 대비가 4.5:1 이상이어야 합니다")
		}
	}
	return nil
}
