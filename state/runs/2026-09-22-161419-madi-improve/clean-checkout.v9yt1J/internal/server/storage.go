package server

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

//go:embed storage_schema.sql
var storageSchema string

type storageProvider struct {
	ID, WorkspaceID, Name, Kind string
	Config                      map[string]any
	Enabled                     bool
}
type storedObject struct {
	ProviderID string `json:"storage_provider_id"`
	Key        string `json:"object_key"`
	Path       string `json:"path"`
	Checksum   string `json:"checksum_sha256"`
	Size       int64  `json:"size"`
}

func (s *Server) migrateStorage(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, storageSchema)
	return e
}
func (s *Server) storageProvider(ctx context.Context, id string) (storageProvider, error) {
	var p storageProvider
	var raw []byte
	e := s.DB.QueryRow(ctx, "SELECT id::text,coalesce(workspace_id::text,''),name,kind,config,enabled FROM storage_providers WHERE id=$1", id).Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Kind, &raw, &p.Enabled)
	if e != nil {
		return p, errors.New("저장소 연결 설정을 찾을 수 없습니다")
	}
	if e = json.Unmarshal(raw, &p.Config); e != nil {
		return p, e
	}
	for _, key := range []string{"access_key", "secret_key", "session_token"} {
		if value := str(p.Config, key); value != "" {
			plain, e := s.decrypt(value)
			if e != nil {
				return p, errors.New("저장소 자격 증명을 복호화할 수 없습니다")
			}
			p.Config[key] = plain
		}
	}
	return p, nil
}
func (s *Server) resolveStorage(ctx context.Context, wid string) (storageProvider, error) {
	var id string
	e := s.DB.QueryRow(ctx, `SELECT coalesce((SELECT provider_id::text FROM storage_assignments WHERE workspace_id=NULLIF($1,'')::uuid),(SELECT provider_id::text FROM storage_settings WHERE id=1),'')`, wid).Scan(&id)
	if e != nil {
		return storageProvider{}, e
	}
	if id != "" {
		p, e := s.storageProvider(ctx, id)
		if e != nil {
			return p, e
		}
		if !p.Enabled {
			return p, errors.New("현재 쓰기 저장소가 비활성화되어 있습니다")
		}
		if p.WorkspaceID != "" && p.WorkspaceID != wid {
			return p, errors.New("다른 워크스페이스 저장소는 사용할 수 없습니다")
		}
		return p, nil
	}
	settings, e := s.settings(ctx)
	return storageProvider{Kind: "local", Enabled: true, Config: map[string]any{"root": str(settings, "storage_path")}}, e
}
func validObjectKey(key string) bool {
	return key != "" && len(key) < 1000 && path.Clean(key) == key && !strings.HasPrefix(key, "/") && filepath.IsLocal(key) && !strings.Contains(key, "\\")
}
func validateStorageConfig(kind string, c map[string]any) error {
	allowed := map[string]bool{"root": true, "endpoint": true, "bucket": true, "prefix": true, "region": true, "access_key": true, "secret_key": true, "session_token": true, "allow_http": true, "insecure_tls": true, "ca_pem": true}
	for key, value := range c {
		if strings.HasSuffix(key, "_configured") {
			continue
		}
		if !allowed[key] {
			return errors.New("지원하지 않는 저장소 설정 항목입니다")
		}
		if key == "allow_http" || key == "insecure_tls" {
			if _, ok := value.(bool); !ok {
				return errors.New("TLS·HTTP 옵션은 체크박스 값이어야 합니다")
			}
		} else {
			v, ok := value.(string)
			if !ok || len(v) > 64<<10 {
				return errors.New("저장소 설정은 64KB 이하 문자열이어야 합니다")
			}
		}
	}
	if kind == "local" {
		root := str(c, "root")
		if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" {
			return errors.New("로컬 저장 경로는 /가 아닌 절대 경로여야 합니다")
		}
		return nil
	}
	if kind != "s3" {
		return errors.New("로컬 또는 S3 저장소를 선택하세요")
	}
	u, e := url.Parse(str(c, "endpoint"))
	if e != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("S3 엔드포인트는 경로·쿼리가 없는 HTTP(S) 주소입니다")
	}
	if u.Scheme == "http" && !boolean(c, "allow_http") {
		return errors.New("사내 HTTP 사용을 명시적으로 허용하세요. 기본은 HTTPS입니다")
	}
	if str(c, "bucket") == "" || strings.ContainsAny(str(c, "bucket"), "/\\ ") {
		return errors.New("S3 버킷 이름을 확인하세요")
	}
	if str(c, "access_key") == "" || str(c, "secret_key") == "" {
		return errors.New("S3 액세스 키와 비밀 키가 필요합니다")
	}
	prefix := str(c, "prefix")
	if prefix != "" && !validObjectKey(prefix) {
		return errors.New("S3 접두사는 상대 경로이며 .. 또는 끝 슬래시를 사용할 수 없습니다")
	}
	if pem := str(c, "ca_pem"); pem != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(pem)) {
			return errors.New("CA 인증서 PEM 형식을 확인하세요")
		}
	}
	return nil
}
func storageS3(p storageProvider) (*minio.Client, error) {
	if e := validateStorageConfig(p.Kind, p.Config); e != nil {
		return nil, e
	}
	u, _ := url.Parse(str(p.Config, "endpoint"))
	roots, _ := x509.SystemCertPool()
	if roots == nil {
		roots = x509.NewCertPool()
	}
	if pem := str(p.Config, "ca_pem"); pem != "" {
		roots.AppendCertsFromPEM([]byte(pem))
	}
	transport := &http.Transport{DialContext: webhookDial, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, InsecureSkipVerify: boolean(p.Config, "insecure_tls")}, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 60 * time.Second, DisableKeepAlives: true}
	region := str(p.Config, "region")
	if region == "" {
		region = "us-east-1"
	}
	return minio.New(u.Host, &minio.Options{Creds: credentials.NewStaticV4(str(p.Config, "access_key"), str(p.Config, "secret_key"), str(p.Config, "session_token")), Secure: u.Scheme == "https", Region: region, Transport: transport, BucketLookup: minio.BucketLookupPath, MaxRetries: 3})
}
func objectMap(m map[string]any) storedObject {
	var size int64
	switch v := m["size"].(type) {
	case float64:
		size = int64(v)
	case int:
		size = int64(v)
	case int64:
		size = v
	case json.Number:
		size, _ = v.Int64()
	}
	return storedObject{ProviderID: str(m, "storage_provider_id"), Key: str(m, "object_key"), Path: str(m, "path"), Checksum: str(m, "checksum_sha256"), Size: size}
}
func (s *Server) putStoredObject(ctx context.Context, p storageProvider, key string, input io.Reader, maxBytes int64, contentType string) (storedObject, error) {
	result := storedObject{ProviderID: p.ID, Key: key}
	if !validObjectKey(key) {
		return result, errors.New("안전하지 않은 저장소 객체 키입니다")
	}
	tmp, e := os.CreateTemp("", "madi-object-*")
	if e != nil {
		return result, e
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	hash := sha256.New()
	n, e := io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(&contextVaultReader{ctx: ctx, reader: input}, maxBytes+1))
	if e != nil || n > maxBytes {
		return result, errors.New("파일 저장 중 오류 또는 크기 제한 초과")
	}
	result.Size = n
	result.Checksum = hex.EncodeToString(hash.Sum(nil))
	if _, e = tmp.Seek(0, 0); e != nil {
		return result, e
	}
	if p.Kind == "local" {
		root := str(p.Config, "root")
		if e = validateStorageConfig("local", p.Config); e != nil {
			return result, e
		}
		if e = os.MkdirAll(root, 0700); e != nil {
			return result, errors.New("로컬 저장 경로의 쓰기 권한을 확인하세요")
		}
		dir, e := os.OpenRoot(root)
		if e != nil {
			return result, e
		}
		defer dir.Close()
		if parent := path.Dir(key); parent != "." {
			if e = dir.MkdirAll(parent, 0700); e != nil {
				return result, e
			}
		}
		output, e := dir.OpenFile(key, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return result, errors.New("기존 객체를 덮어쓰지 않습니다. 저장 경로를 확인하세요")
		}
		_, e = io.Copy(output, tmp)
		if e == nil {
			e = output.Sync()
		}
		closeErr := output.Close()
		if e == nil {
			e = closeErr
		}
		if e != nil {
			_ = dir.Remove(key)
			return result, e
		}
		result.Path = filepath.Join(root, filepath.FromSlash(key))
		return result, nil
	}
	client, e := storageS3(p)
	if e != nil {
		return result, e
	}
	objectKey := key
	if prefix := str(p.Config, "prefix"); prefix != "" {
		objectKey = prefix + "/" + key
	}
	_, e = client.PutObject(ctx, str(p.Config, "bucket"), objectKey, tmp, n, minio.PutObjectOptions{ContentType: contentType, DisableMultipart: true, SendContentMd5: true, UserMetadata: map[string]string{"madi-sha256": result.Checksum}})
	if e != nil {
		return result, errors.New("S3 업로드에 실패했습니다. 주소·권한·TLS 설정을 확인하세요")
	}
	return result, nil
}
func (s *Server) openStoredObject(ctx context.Context, o storedObject) (io.ReadCloser, error) {
	if o.ProviderID == "" {
		return openLegacyObject(o)
	}
	p, e := s.storageProvider(ctx, o.ProviderID)
	if e != nil {
		return nil, e
	}
	if !validObjectKey(o.Key) {
		return nil, errors.New("객체 키를 확인하세요")
	}
	if p.Kind == "local" {
		root, e := os.OpenRoot(str(p.Config, "root"))
		if e != nil {
			return nil, e
		}
		f, e := root.Open(o.Key)
		root.Close()
		return f, e
	}
	client, e := storageS3(p)
	if e != nil {
		return nil, e
	}
	key := o.Key
	if prefix := str(p.Config, "prefix"); prefix != "" {
		key = prefix + "/" + key
	}
	object, e := client.GetObject(ctx, str(p.Config, "bucket"), key, minio.GetObjectOptions{})
	if e != nil {
		return nil, errors.New("S3 객체를 읽을 수 없습니다")
	}
	if _, e = object.Stat(); e != nil {
		object.Close()
		return nil, errors.New("S3 객체가 없거나 읽기 권한이 없습니다")
	}
	return object, nil
}
func openLegacyObject(o storedObject) (io.ReadCloser, error) {
	if !filepath.IsAbs(o.Path) || filepath.Clean(o.Path) != o.Path || !validID(strings.TrimSuffix(filepath.Base(o.Path), ".zip")) {
		return nil, errors.New("기존 첨부파일 경로를 확인하세요")
	}
	info, e := os.Lstat(o.Path)
	if e != nil || !info.Mode().IsRegular() {
		return nil, errors.New("일반 첨부파일이 없거나 안전하지 않습니다")
	}
	return os.Open(o.Path)
}

// Materialize before writing response headers, allowing checksum failures and
// object-store errors to return a proper error response; also supports HTTP Range.
func (s *Server) materializeObject(ctx context.Context, o storedObject, maxBytes int64) (*os.File, error) {
	input, e := s.openStoredObject(ctx, o)
	if e != nil {
		return nil, e
	}
	defer input.Close()
	tmp, e := os.CreateTemp("", "madi-download-*")
	if e != nil {
		return nil, e
	}
	good := false
	defer func() {
		if !good {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	hash := sha256.New()
	n, e := io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(&contextVaultReader{ctx: ctx, reader: input}, maxBytes+1))
	if e != nil || n > maxBytes || (o.Size >= 0 && n != o.Size) {
		return nil, errors.New("첨부파일 크기 또는 읽기 무결성 검증에 실패했습니다")
	}
	if o.Checksum != "" && o.Checksum != hex.EncodeToString(hash.Sum(nil)) {
		return nil, errors.New("첨부파일 SHA-256 검증에 실패했습니다")
	}
	if _, e = tmp.Seek(0, 0); e != nil {
		return nil, e
	}
	good = true
	return tmp, nil
}
func (s *Server) deleteStoredObject(ctx context.Context, o storedObject) error {
	if o.ProviderID == "" {
		if !filepath.IsAbs(o.Path) || filepath.Clean(o.Path) != o.Path {
			return errors.New("안전하지 않은 객체 경로")
		}
		base := strings.TrimSuffix(filepath.Base(o.Path), ".zip")
		if !validID(base) {
			return errors.New("madi UUID 객체만 삭제할 수 있습니다")
		}
		info, e := os.Lstat(o.Path)
		if errors.Is(e, os.ErrNotExist) {
			return nil
		}
		if e != nil || !info.Mode().IsRegular() {
			return errors.New("일반 파일이 아니므로 보존했습니다")
		}
		return os.Remove(o.Path)
	}
	p, e := s.storageProvider(ctx, o.ProviderID)
	if e != nil {
		return e
	}
	if !validObjectKey(o.Key) || !validID(strings.TrimSuffix(path.Base(o.Key), ".zip")) {
		return errors.New("madi UUID 객체만 삭제할 수 있습니다")
	}
	if p.Kind == "local" {
		root, e := os.OpenRoot(str(p.Config, "root"))
		if e != nil {
			return e
		}
		defer root.Close()
		e = root.Remove(o.Key)
		if errors.Is(e, os.ErrNotExist) {
			return nil
		}
		return e
	}
	client, e := storageS3(p)
	if e != nil {
		return e
	}
	key := o.Key
	if prefix := str(p.Config, "prefix"); prefix != "" {
		key = prefix + "/" + key
	}
	if e = client.RemoveObject(ctx, str(p.Config, "bucket"), key, minio.RemoveObjectOptions{}); e != nil {
		return errors.New("S3 객체 삭제에 실패했습니다")
	}
	return nil
}
