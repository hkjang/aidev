package server

import (
	"strings"
	"testing"
)

func TestProfilePreferencesStrictValuesAndMerge(t *testing.T) {
	s, c, ctx, p, _ := jobTestFixture(t)
	valid := map[string]any{"font_family": "serif", "page_width": "full", "spell_check": false, "code_theme": "dark", "date_format": "iso", "font_size": 24, "sidebar_width": 420, "theme": "light", "timezone": "Asia/Tokyo", "editor_mode": "source"}
	result := testJSONObject(t, c.request("PUT", "/api/v1/profile", map[string]any{"expected_user_id": p.ID, "preferences": valid}, 200))
	if result["preferences"].(map[string]any)["spell_check"] != false {
		t.Fatal("false preference lost")
	}
	for _, preset := range []string{"personal", "wiki", "database", "operations"} {
		c.request("PUT", "/api/v1/profile", map[string]any{"preferences": map[string]any{"nav_preset": preset, "navigation_advanced": false, "document_panel": "versions", "document_panel_open": true}}, 200)
	}
	for _, density := range []string{"comfortable", "compact", "relaxed"} {
		c.request("PUT", "/api/v1/profile", map[string]any{"preferences": map[string]any{"density": density, "mobile_table_view": "cards"}}, 200)
	}
	c.request("PUT", "/api/v1/profile", map[string]any{"preferences": map[string]any{"mobile_table_view": "table"}}, 200)
	for _, invalid := range []map[string]any{{"density": "tiny"}, {"density": nil}, {"density": 1}, {"mobile_table_view": "html"}, {"mobile_table_view": true}} {
		c.request("PUT", "/api/v1/profile", map[string]any{"preferences": invalid}, 400)
	}
	for _, invalid := range []map[string]any{{"nav_preset": "admin"}, {"document_panel": "html"}, {"navigation_advanced": "false"}, {"document_panel_open": 0}} {
		c.request("PUT", "/api/v1/profile", map[string]any{"preferences": invalid}, 400)
	}
	result = testJSONObject(t, c.request("PUT", "/api/v1/profile", map[string]any{"preferences": map[string]any{"navigation_pins": []string{"/app/tasks", "/app/tasks", "/app/graph"}}}, 200))
	if pins := result["preferences"].(map[string]any)["navigation_pins"].([]any); len(pins) != 2 {
		t.Fatal("navigation pins not deduplicated", pins)
	}
	for _, pins := range []any{[]string{"https://outside.example"}, []string{"/admin/users"}, []string{"/app/not-registered"}, "/app/tasks"} {
		c.request("PUT", "/api/v1/profile", map[string]any{"preferences": map[string]any{"navigation_pins": pins}}, 400)
	}
	for _, patch := range []map[string]any{{"font_family": "https://remote/font"}, {"font_family": nil}, {"page_width": "2000px"}, {"spell_check": "false"}, {"code_theme": true}, {"date_format": "yyyy"}, {"font_size": "18"}, {"font_size": 15}, {"font_size": 24.1}, {"font_size": 25}, {"sidebar_width": 421}, {"theme": "system"}, {"timezone": "invalid"}, {"editor_mode": "html"}, {"language": "en"}, {"keyboard_shortcuts": map[string]any{"save": "Mod+W"}}, {"keyboard_shortcuts": map[string]any{"save": "Mod+K"}}, {"keyboard_shortcuts": map[string]any{"unknown": "Mod+E"}}} {
		c.request("PUT", "/api/v1/profile", map[string]any{"name": "수정되면 안 되는 이름", "preferences": patch}, 400)
	}
	c.request("PUT", "/api/v1/profile", map[string]any{"preferences": "bad"}, 400)
	result = testJSONObject(t, c.request("GET", "/api/v1/auth/me", nil, 200))
	if str(result, "name") == "수정되면 안 되는 이름" || result["preferences"].(map[string]any)["font_family"] != "serif" {
		t.Fatal("invalid transaction partially committed")
	}
	c.request("PUT", "/api/v1/profile", map[string]any{"preferences": map[string]any{"keyboard_shortcuts": map[string]any{"save": "Mod+Shift+E"}}}, 200)
	c.request("PUT", "/api/v1/profile", map[string]any{"preferences": map[string]any{"extension_state": strings.Repeat("x", 31000)}}, 200)
	c.request("PUT", "/api/v1/profile", map[string]any{"preferences": map[string]any{"another_extension": strings.Repeat("y", 2000)}}, 400)
	var prefs map[string]any
	if err := s.DB.QueryRow(ctx, "SELECT preferences FROM users WHERE id=$1", p.ID).Scan(&prefs); err != nil {
		t.Fatal(err)
	}
	if _, ok := prefs["another_extension"]; ok {
		t.Fatal("merge exceeded total size")
	}
}
