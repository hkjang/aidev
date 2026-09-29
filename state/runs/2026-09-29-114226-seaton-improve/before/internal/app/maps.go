package app

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (s *Server) listBuildings(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT id,name,code,COALESCE(address,'') FROM buildings ORDER BY name`)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, name, code, address string
		if rows.Scan(&id, &name, &code, &address) == nil {
			items = append(items, map[string]any{"id": id, "name": name, "code": code, "address": address})
		}
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (s *Server) createBuilding(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name    string `json:"name"`
		Code    string `json:"code"`
		Address string `json:"address"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	if in.Name == "" || in.Code == "" {
		writeError(w, 400, "required_fields", "건물명과 코드는 필수입니다")
		return
	}
	id := newID()
	_, err := s.db.Exec(r.Context(), `INSERT INTO buildings(id,name,code,address) VALUES($1,$2,$3,NULLIF($4,''))`, id, in.Name, in.Code, in.Address)
	if err != nil {
		writeError(w, 409, "building_conflict", "이미 사용 중인 건물 코드입니다")
		return
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "building.create", "building", id, r.RemoteAddr, in)
	writeJSON(w, 201, map[string]string{"id": id})
}

func (s *Server) listFloors(w http.ResponseWriter, r *http.Request) {
	buildingID := r.URL.Query().Get("buildingId")
	rows, err := s.db.Query(r.Context(), `SELECT f.id,f.building_id,f.name,f.code,f.sort_order,b.name FROM floors f JOIN buildings b ON b.id=f.building_id WHERE ($1='' OR f.building_id=$1) ORDER BY b.name,f.sort_order,f.name`, buildingID)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, bid, name, code, bname string
		var order int
		if rows.Scan(&id, &bid, &name, &code, &order, &bname) == nil {
			items = append(items, map[string]any{"id": id, "buildingId": bid, "buildingName": bname, "name": name, "code": code, "sortOrder": order})
		}
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (s *Server) createFloor(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BuildingID string `json:"buildingId"`
		Name       string `json:"name"`
		Code       string `json:"code"`
		SortOrder  int    `json:"sortOrder"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	if in.BuildingID == "" || in.Name == "" || in.Code == "" {
		writeError(w, 400, "required_fields", "건물, 층 이름, 코드는 필수입니다")
		return
	}
	id := newID()
	_, err := s.db.Exec(r.Context(), `INSERT INTO floors(id,building_id,name,code,sort_order) VALUES($1,$2,$3,$4,$5)`, id, in.BuildingID, in.Name, in.Code, in.SortOrder)
	if err != nil {
		writeError(w, 409, "floor_conflict", "층 정보가 중복되었거나 건물이 없습니다")
		return
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "floor.create", "floor", id, r.RemoteAddr, in)
	writeJSON(w, 201, map[string]string{"id": id})
}

func (s *Server) listFloorMaps(w http.ResponseWriter, r *http.Request) {
	floorID := r.URL.Query().Get("floorId")
	// PDF는 래스터 미리보기의 픽셀 크기가 좌석 오버레이의 기준이 되므로 그것을 우선 노출한다.
	rows, err := s.db.Query(r.Context(), `SELECT m.id,m.floor_id,m.version,m.file_name,m.content_type,COALESCE(m.preview_width,m.width),COALESCE(m.preview_height,m.height),m.status,m.is_active,m.created_at,f.name,b.name,stats.seat_count,stats.review_count,m.grid
	FROM floor_maps m JOIN floors f ON f.id=m.floor_id JOIN buildings b ON b.id=f.building_id
	LEFT JOIN LATERAL (SELECT COUNT(*) seat_count,COUNT(*) FILTER(WHERE confidence IS NOT NULL AND confidence < .95) review_count FROM seats s WHERE s.floor_map_id=m.id) stats ON true
	WHERE ($1='' OR m.floor_id=$1) ORDER BY m.created_at DESC`, floorID)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, fid, version, name, ct, status, fname, bname string
		var width, height *int
		var active bool
		var seatCount, reviewCount int
		var created any
		var gridRaw []byte
		if rows.Scan(&id, &fid, &version, &name, &ct, &width, &height, &status, &active, &created, &fname, &bname, &seatCount, &reviewCount, &gridRaw) == nil {
			items = append(items, map[string]any{"id": id, "floorId": fid, "version": version, "fileName": name, "contentType": ct, "width": width, "height": height, "status": status, "active": active, "createdAt": created, "floorName": fname, "buildingName": bname, "seatCount": seatCount, "reviewCount": reviewCount, "contentUrl": "/api/v1/floor-maps/" + id + "/content", "previewUrl": "/api/v1/floor-maps/" + id + "/preview", "overlayReady": width != nil && height != nil, "grid": parseSeatGrid(gridRaw)})
		}
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (s *Server) uploadFloorMap(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 30<<20)
	if err := r.ParseMultipartForm(30 << 20); err != nil {
		writeError(w, 400, "upload_too_large", "도면 파일은 25MB 이하여야 합니다")
		return
	}
	floorID := strings.TrimSpace(r.FormValue("floorId"))
	version := strings.TrimSpace(r.FormValue("version"))
	file, header, err := r.FormFile("file")
	if err != nil || floorID == "" || version == "" {
		writeError(w, 400, "required_fields", "층, 버전, 도면 파일은 필수입니다")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 25<<20+1))
	if err != nil || len(data) > 25<<20 {
		writeError(w, 400, "upload_too_large", "도면 파일은 25MB 이하여야 합니다")
		return
	}
	ct := http.DetectContentType(data)
	allowed := map[string]bool{"image/png": true, "image/jpeg": true, "application/pdf": true}
	if !allowed[ct] {
		writeError(w, 400, "unsupported_map", "PNG, JPG, PDF 도면만 업로드할 수 있습니다")
		return
	}
	var width, height *int
	if strings.HasPrefix(ct, "image/") {
		if cfg, _, e := image.DecodeConfig(bytes.NewReader(data)); e == nil {
			width = &cfg.Width
			height = &cfg.Height
		}
	}
	// PDF는 업로드 시점에 한 번 래스터화해 좌석 오버레이 배경으로 재사용한다.
	// 변환기가 없어도 업로드 자체는 성공시키고, 미리보기는 최초 요청 시 다시 시도한다.
	var previewData []byte
	var previewWidth, previewHeight *int
	if ct == "application/pdf" {
		if raster, pw, ph, e := rasterizePDF(r.Context(), data); e == nil {
			previewData, previewWidth, previewHeight = raster, &pw, &ph
		} else {
			s.logger.Warn("도면 미리보기 생성 실패", "error", e)
		}
	}
	id := newID()
	u, _ := userFrom(r)
	_, err = s.db.Exec(r.Context(), `INSERT INTO floor_maps(id,floor_id,version,file_name,content_type,file_data,width,height,preview_data,preview_width,preview_height,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, id, floorID, version, safeFilename(header), ct, data, width, height, previewData, previewWidth, previewHeight, u.ID)
	if err != nil {
		writeError(w, 409, "map_conflict", "동일한 층과 버전의 도면이 이미 있습니다")
		return
	}
	s.audit(r.Context(), u.ID, "floor_map.upload", "floor_map", id, r.RemoteAddr, map[string]any{"file": header.Filename, "size": len(data), "overlayReady": previewWidth != nil || width != nil})
	writeJSON(w, 201, map[string]string{"id": id})
}

func safeFilename(h *multipart.FileHeader) string {
	name := strings.ReplaceAll(h.Filename, "\\", "/")
	parts := strings.Split(name, "/")
	return parts[len(parts)-1]
}

func (s *Server) mapContent(w http.ResponseWriter, r *http.Request) {
	var data []byte
	var ct, name string
	err := s.db.QueryRow(r.Context(), `SELECT file_data,content_type,file_name FROM floor_maps WHERE id=$1`, chi.URLParam(r, "mapID")).Scan(&data, &ct, &name)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", `inline; filename="`+strings.ReplaceAll(name, `"`, "")+`"`)
	w.Header().Set("Cache-Control", "private, max-age=300")
	_, _ = w.Write(data)
}

// mapPreview는 좌석 오버레이의 배경으로 쓸 래스터 이미지를 돌려준다.
// 이미지 도면은 원본을, PDF는 저장된 미리보기를 내보내고, 미리보기가 없으면
// 이번 요청에서 한 번 만들어 저장한다(기존에 올라간 PDF 도면 보정용).
func (s *Server) mapPreview(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "mapID")
	var ct, name string
	var preview []byte
	err := s.db.QueryRow(r.Context(), `SELECT content_type,file_name,preview_data FROM floor_maps WHERE id=$1`, id).Scan(&ct, &name, &preview)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	if ct != "application/pdf" {
		s.mapContent(w, r)
		return
	}
	preview, err = s.ensurePreview(r.Context(), id, preview)
	if err != nil {
		writeError(w, 422, "preview_unavailable", "PDF 미리보기를 만들지 못했습니다")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", `inline; filename="`+strings.ReplaceAll(name, `"`, "")+`.png"`)
	w.Header().Set("Cache-Control", "private, max-age=300")
	_, _ = w.Write(preview)
}

// unpublishFloorMap은 게시를 내린다.
//
// 게시 중인 도면은 지울 수 없는데, 그 층의 유일한 버전이면 다른 버전을 게시해
// 밀어낼 수도 없어 잘못 올린 도면이 영영 남는다. 게시를 내리면 좌석맵에서는
// 사라지고 좌석과 이력은 그대로 남으므로, 그 뒤에 지울지 다시 게시할지 고를 수
// 있다.
func (s *Server) unpublishFloorMap(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "mapID")
	tag, err := s.db.Exec(r.Context(), `UPDATE floor_maps SET is_active=false,status='archived',published_at=NULL WHERE id=$1 AND is_active`, id)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusConflict, "map_not_published", "게시 중인 도면이 아닙니다")
		return
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "floor_map.unpublish", "floor_map", id, r.RemoteAddr, nil)
	w.WriteHeader(http.StatusNoContent)
}

// deleteFloorMap은 잘못 올린 도면 버전을 지운다.
//
// 게시 중인 도면과 이력이 남은 도면은 지우지 않는다. 좌석이 사라지면 변경 이력의
// 좌석 참조가 비어 "누가 어디서 어디로 옮겼는지"를 잃기 때문이다. 실수로 올린
// 버전은 배정도 이력도 없으므로 이 조건에 걸리지 않는다.
func (s *Server) deleteFloorMap(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "mapID")
	var active bool
	if err := s.db.QueryRow(r.Context(), `SELECT is_active FROM floor_maps WHERE id=$1`, id).Scan(&active); err != nil {
		notFoundOrServer(w, err)
		return
	}
	if active {
		writeError(w, http.StatusConflict, "map_published", "게시 중인 도면은 삭제할 수 없습니다. 다른 버전을 게시한 뒤 삭제하세요")
		return
	}
	var used int
	err := s.db.QueryRow(r.Context(), `SELECT count(*) FROM seats s
                WHERE s.floor_map_id=$1 AND (
                        EXISTS(SELECT 1 FROM seat_assignments a WHERE a.seat_id=s.id AND a.ended_at IS NULL)
                        OR EXISTS(SELECT 1 FROM seat_history h WHERE h.previous_seat_id=s.id OR h.new_seat_id=s.id))`, id).Scan(&used)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	if used > 0 {
		writeError(w, http.StatusConflict, "map_in_use", "배정이나 변경 이력이 있는 도면은 삭제할 수 없습니다")
		return
	}
	if _, err := s.db.Exec(r.Context(), `DELETE FROM floor_maps WHERE id=$1`, id); err != nil {
		notFoundOrServer(w, err)
		return
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "floor_map.delete", "floor_map", id, r.RemoteAddr, nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) publishFloorMap(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "mapID")
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var floorID string
	if err = tx.QueryRow(r.Context(), `SELECT floor_id FROM floor_maps WHERE id=$1`, id).Scan(&floorID); err != nil {
		notFoundOrServer(w, err)
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE floor_maps SET is_active=false,status=CASE WHEN status='published' THEN 'archived' ELSE status END WHERE floor_id=$1`, floorID)
	if err == nil {
		_, err = tx.Exec(r.Context(), `UPDATE floor_maps SET is_active=true,status='published',published_at=now() WHERE id=$1`, id)
	}
	if err != nil || tx.Commit(r.Context()) != nil {
		notFoundOrServer(w, err)
		return
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "floor_map.publish", "floor_map", id, r.RemoteAddr, nil)
	w.WriteHeader(204)
}

func parseFloat(s string, fallback float64) float64 {
	v, e := strconv.ParseFloat(s, 64)
	if e != nil {
		return fallback
	}
	return v
}
