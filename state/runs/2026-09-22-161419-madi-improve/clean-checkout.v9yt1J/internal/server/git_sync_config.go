package server

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
)

type gitSyncConnection struct {
	ID, WorkspaceID, OwnerID, SpaceID, Name, LastRemoteCommit string
	Config                                                    map[string]any
	Enabled                                                   bool
	Revision                                                  int64
}

var gitSyncPathPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
var gitSyncUserPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)

func gitSyncValidateConfig(cfg map[string]any) error {
	for key := range cfg {
		if !oneOf(key, "url", "branch", "prefix", "username", "password", "private_key", "private_key_password", "host_key", "ca_pem") {
			return errors.New("지원하지 않는 Git 연결 설정입니다")
		}
		if _, ok := cfg[key].(string); !ok {
			return errors.New("Git 연결 설정은 문자열이어야 합니다")
		}
	}
	raw := str(cfg, "url")
	u, e := url.Parse(raw)
	if e != nil || !oneOf(u.Scheme, "https", "ssh") || u.Hostname() == "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.RawPath != "" || len(raw) > 1500 {
		return errors.New("Git 연결은 명시적인 HTTPS 또는 SSH URL이어야 합니다")
	}
	if !gitSyncPathPattern.MatchString(u.Path) || !strings.HasPrefix(u.Path, "/") || path.Clean(u.Path) != u.Path || u.Path == "/" {
		return errors.New("Git 저장소 경로를 확인하세요")
	}
	if u.Port() != "" {
		port, e := strconv.Atoi(u.Port())
		if e != nil || port < 1 || port > 65535 {
			return errors.New("Git 서버 포트를 확인하세요")
		}
	}
	if u.Scheme == "https" && u.User != nil {
		return errors.New("Git 주소 안에 계정이나 암호를 넣지 마세요")
	}
	if u.Scheme == "ssh" {
		if u.User == nil || !gitSyncUserPattern.MatchString(u.User.Username()) {
			return errors.New("SSH 주소에 실행 계정이 필요합니다")
		}
		if _, found := u.User.Password(); found {
			return errors.New("SSH URL에 암호를 넣지 마세요")
		}
		if str(cfg, "private_key") == "" || str(cfg, "host_key") == "" {
			return errors.New("SSH 개인 키와 고정 서버 공개 키가 모두 필요합니다")
		}
	}
	branch := str(cfg, "branch")
	if len(branch) > 100 || strings.HasPrefix(branch, "refs/") || branch == "HEAD" || plumbing.NewBranchReferenceName(branch).Validate() != nil {
		return errors.New("고정 Git 브랜치 이름을 확인하세요")
	}
	prefix := str(cfg, "prefix")
	if prefix == "" || len(prefix) > 200 || !gitSyncPathPattern.MatchString(prefix) || strings.HasPrefix(prefix, "/") || path.Clean(prefix) != prefix {
		return errors.New("Git 저장소 안의 전용 상대 폴더를 지정하세요")
	}
	for _, part := range strings.Split(prefix, "/") {
		if strings.EqualFold(part, ".git") || part == "." || part == ".." {
			return errors.New("Git 내부 설정 경로는 사용할 수 없습니다")
		}
	}
	for _, key := range []string{"username", "password", "private_key", "private_key_password", "host_key", "ca_pem"} {
		if len(str(cfg, key)) > 64<<10 || strings.ContainsRune(str(cfg, key), 0) {
			return errors.New("Git 인증 설정의 크기·문자를 확인하세요")
		}
	}
	return nil
}

func gitSyncResolve(ctx context.Context, raw string, allowed []string, allowPrivate bool) (*url.URL, []netip.Addr, error) {
	u, e := url.Parse(raw)
	if e != nil {
		return nil, nil, errors.New("Git 주소를 읽을 수 없습니다")
	}
	accepted := false
	for _, host := range allowed {
		if strings.EqualFold(host, u.Hostname()) {
			accepted = true
		}
	}
	if !accepted {
		return nil, nil, errors.New("관리자가 허용한 Git 호스트가 아닙니다")
	}
	resolved, e := net.DefaultResolver.LookupNetIP(ctx, "ip", u.Hostname())
	if e != nil || len(resolved) == 0 {
		return nil, nil, errors.New("Git 호스트 주소를 확인할 수 없습니다")
	}
	for i, ip := range resolved {
		ip = ip.Unmap()
		resolved[i] = ip
		if !ip.IsValid() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || (!allowPrivate && (ip.IsPrivate() || ip.IsLoopback())) {
			return nil, nil, errors.New("Git 호스트가 네트워크 허용 정책을 벗어났습니다")
		}
	}
	return u, resolved, nil
}
