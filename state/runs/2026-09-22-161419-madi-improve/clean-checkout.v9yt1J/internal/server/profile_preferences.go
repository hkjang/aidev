package server

import (
	"fmt"
	"math"
	"regexp"
)

var profileChordPattern = regexp.MustCompile(`^Mod\+(Alt\+)?(Shift\+)?([A-Z0-9/]|Enter)$`)

func validateProfilePreferences(v map[string]any) error {
	if len(jsonValue(v)) > 32000 {
		return fmt.Errorf("개인 설정은 32KB 이하여야 합니다")
	}
	enums := map[string][]string{"theme": {"light", "dark"}, "font_family": {"sans", "system", "serif"}, "page_width": {"standard", "wide", "full"}, "code_theme": {"auto", "light", "dark"}, "date_format": {"ko", "iso", "long"}, "editor_mode": {"edit", "source", "preview"}, "timezone": {"Asia/Seoul", "UTC", "Asia/Tokyo", "America/New_York"}, "language": {"ko"}, "document_filter": {"all", "private", "shared"}}
	enums["nav_preset"] = []string{"personal", "wiki", "database", "operations"}
	enums["document_panel"] = []string{"backlinks", "properties", "comments", "ai", "versions"}
	enums["density"] = []string{"comfortable", "compact", "relaxed"}
	enums["mobile_table_view"] = []string{"cards", "table"}
	for key, allowed := range enums {
		if value, ok := v[key]; ok {
			word, valid := value.(string)
			if !valid || !oneOf(word, allowed...) {
				return fmt.Errorf("개인 설정 %s의 선택값을 확인하세요", key)
			}
		}
	}
	for key, bounds := range map[string][2]float64{"font_size": {16, 24}, "sidebar_width": {220, 420}} {
		if value, ok := v[key]; ok {
			n, valid := value.(float64)
			if !valid || math.IsNaN(n) || math.IsInf(n, 0) || n != math.Trunc(n) || n < bounds[0] || n > bounds[1] {
				return fmt.Errorf("개인 설정 %s는 %.0f~%.0f 범위의 정수여야 합니다", key, bounds[0], bounds[1])
			}
		}
	}
	for _, key := range []string{"spell_check", "sidebar_collapsed", "navigation_advanced", "document_panel_open"} {
		if value, ok := v[key]; ok {
			if _, valid := value.(bool); !valid {
				return fmt.Errorf("개인 설정 %s는 켜기 또는 끄기여야 합니다", key)
			}
		}
	}
	if raw, ok := v["navigation_pins"]; ok {
		pins, valid := raw.([]any)
		if !valid || len(pins) > 12 {
			return fmt.Errorf("고정 도구는 최대 12개 경로의 배열이어야 합니다")
		}
		seen := map[string]bool{}
		unique := []any{}
		for _, pin := range pins {
			path, ok := pin.(string)
			if ok && oneOf(path, "/app/worksets", "/app/system-status", "/app/knowledge-packages", "/app/knowledge-impact", "/app/knowledge-proposals", "/app/knowledge-time", "/app/knowledge-distribution", "/app/knowledge-questions", "/app/knowledge-conflicts", "/app/knowledge-paths", "/app/structured-drafts", "/app/access-requests") {
				if !seen[path] {
					unique = append(unique, path)
					seen[path] = true
				}
				continue
			}
			if !ok || !oneOf(path, "/app", "/app/documents", "/app/search", "/app/inbox", "/app/my-work", "/app/favorites", "/app/graph", "/app/databases", "/app/tasks", "/app/spaces", "/app/canvases", "/app/templates", "/app/trash", "/app/ai-history", "/app/search-history", "/app/evidence", "/app/agents", "/app/graph-ai", "/app/knowledge-health", "/app/enterprise", "/app/entities", "/app/import", "/app/export", "/app/migrations", "/app/git-sync", "/app/plugins", "/app/connectors", "/app/data-sources", "/app/automations", "/app/jobs", "/app/approvals", "/app/members", "/app/teams", "/app/organizations", "/app/webhooks", "/app/workspace-settings", "/app/workspace-audit", "/app/workspace-operations", "/app/search-ai-settings", "/app/storage") {
				return fmt.Errorf("고정할 수 없는 도구 경로입니다")
			}
			if !seen[path] {
				unique = append(unique, path)
				seen[path] = true
			}
		}
		v["navigation_pins"] = unique
	}
	if raw, ok := v["keyboard_shortcuts"]; ok {
		entries, valid := raw.(map[string]any)
		if !valid {
			return fmt.Errorf("단축키 설정은 객체여야 합니다")
		}
		defaults := map[string]string{"palette": "Mod+K", "quickOpen": "Mod+P", "search": "Mod+Shift+F", "create": "Mod+N", "help": "Mod+/", "save": "Mod+Enter", "focus": "Mod+Shift+L", "ai": "Mod+Shift+A"}
		for key, value := range entries {
			chord, valid := value.(string)
			if _, known := defaults[key]; !known || !valid || !profileChordPattern.MatchString(chord) || oneOf(chord, "Mod+W", "Mod+R", "Mod+T", "Mod+L", "Mod+Q", "Mod+B", "Mod+I", "Mod+U", "Mod+Z", "Mod+Y", "Mod+Shift+Z", "Mod+Shift+W", "Mod+Shift+T", "Mod+Alt+Q") {
				return fmt.Errorf("사용할 수 없는 단축키입니다")
			}
			defaults[key] = chord
		}
		used := map[string]bool{}
		for _, value := range defaults {
			if used[value] {
				return fmt.Errorf("동일한 단축키를 두 명령에 지정할 수 없습니다")
			}
			used[value] = true
		}
	}
	return nil
}
