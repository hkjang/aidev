package tracking

import (
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// MaxViolations 는 기록의 상한이다. 차단된 요청은 페이지마다 반복되므로 중요한
// 정보는 횟수가 아니라 어느 출처가 막혔는가다 — 서로 다른 출처 100개면 스니펫을
// 고치기에 충분하다.
const MaxViolations = 100

// Violation 은 콘텐츠 보안 정책이 거부한 출처 하나다. 거부한 지시어를 함께
// 두어 화면이 무엇을 허용하면 되는지 말할 수 있게 한다.
type Violation struct {
	Origin    string    `json:"origin"`
	Directive string    `json:"directive"`
	Page      string    `json:"page"`
	Count     int       `json:"count"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
	Allowed   bool      `json:"allowed"`
}

// Recorder 는 브라우저가 신고한 정책 위반을 모은다. 일부러 메모리에만 둔다:
// 신고는 스니펫을 붙이는 사람을 위한 실시간 진단이지 감사 기록이 아니고,
// 데이터베이스 밖에 두어야 브라우저가 마음껏 신고해도 저장소가 자라지 않는다.
type Recorder struct {
	mutex      sync.Mutex
	violations map[string]*Violation
	now        func() time.Time
}

func NewRecorder() *Recorder {
	return &Recorder{violations: make(map[string]*Violation), now: time.Now}
}

// Record 는 차단된 요청 하나를 적는다. 브라우저 확장이나 data: 처럼 http 출처가
// 아닌 것은 허용할 수도 없고 허용해도 소용없으므로 버린다.
func (r *Recorder) Record(blockedURI, directive, page string) {
	origin := originOf(blockedURI)
	if origin == "" {
		return
	}
	directive = strings.TrimSpace(strings.ToLower(directive))
	if index := strings.IndexByte(directive, ' '); index > 0 {
		directive = directive[:index]
	}
	if directive == "" {
		directive = "connect-src"
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	key := directive + " " + origin
	if existing, found := r.violations[key]; found {
		existing.Count++
		existing.LastSeen = r.now()
		existing.Page = page
		return
	}
	if len(r.violations) >= MaxViolations {
		r.evictOldest()
	}
	moment := r.now()
	r.violations[key] = &Violation{Origin: origin, Directive: directive, Page: page, Count: 1, FirstSeen: moment, LastSeen: moment}
}

func (r *Recorder) evictOldest() {
	var oldestKey string
	var oldest time.Time
	for key, violation := range r.violations {
		if oldestKey == "" || violation.LastSeen.Before(oldest) {
			oldestKey, oldest = key, violation.LastSeen
		}
	}
	delete(r.violations, oldestKey)
}

// List 는 차단된 출처를 최근 것부터 돌려준다. 지금 설정이 이미 허용하는 것은
// 표시해 두어, 고친 스니펫이 계속 잔소리하지 않게 한다.
func (r *Recorder) List(config Config) []Violation {
	allowed := make(map[string]struct{})
	scripts, connects, images := config.PolicySources()
	for _, group := range [][]string{scripts, connects, images} {
		for _, origin := range group {
			allowed[strings.ToLower(strings.TrimSuffix(origin, "/"))] = struct{}{}
		}
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	items := make([]Violation, 0, len(r.violations))
	for _, violation := range r.violations {
		copied := *violation
		_, known := allowed[copied.Origin]
		copied.Allowed = known || matchesWildcard(copied.Origin, allowed)
		items = append(items, copied)
	}
	sort.Slice(items, func(first, second int) bool {
		if items[first].LastSeen.Equal(items[second].LastSeen) {
			return items[first].Origin < items[second].Origin
		}
		return items[first].LastSeen.After(items[second].LastSeen)
	})
	return items
}

// Forget 은 기록을 비운다. 스니펫을 고친 뒤 아직도 막히는 것이 있는지 다시
// 보려는 관리자가 누른다.
func (r *Recorder) Forget() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.violations = make(map[string]*Violation)
}

// matchesWildcard 는 https://*.google-analytics.com 같은 정책 항목을 다룬다.
func matchesWildcard(origin string, allowed map[string]struct{}) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	for pattern := range allowed {
		star := strings.Index(pattern, "*.")
		if star < 0 {
			continue
		}
		if strings.HasPrefix(origin, pattern[:star]) && strings.HasSuffix(parsed.Host, pattern[star+1:]) {
			return true
		}
	}
	return false
}
