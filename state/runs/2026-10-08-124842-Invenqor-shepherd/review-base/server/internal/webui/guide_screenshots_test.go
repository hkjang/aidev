package webui

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// The user and administrator guides embed screenshots of this console that
// scripts/capture-guide-screenshots.mjs takes. The three lists - the screens
// the script photographs, the files under docs/assets/guide and the figures
// the two guides reference - drift silently: renaming a screen in the script
// leaves the old picture in the guide, and dropping a section leaves an orphan
// PNG that is shipped with every release. A missing figure only surfaces when
// the PDF is built, which is not part of the test run.
func TestGuideScreenshotsMatchCaptureScript(t *testing.T) {
	root := repositoryRoot(t)

	captured := capturedScreens(t, filepath.Join(root, "scripts", "capture-guide-screenshots.mjs"))
	referenced := referencedFigures(t,
		filepath.Join(root, "docs", "USER_GUIDE.md"),
		filepath.Join(root, "docs", "ADMIN_GUIDE.md"),
	)
	stored := storedScreenshots(t, filepath.Join(root, "docs", "assets", "guide"))

	if diff := difference(captured, referenced); len(diff) > 0 {
		t.Errorf("the capture script photographs screens no guide embeds: %v", diff)
	}
	if diff := difference(referenced, captured); len(diff) > 0 {
		t.Errorf("the guides embed figures the capture script does not take: %v", diff)
	}
	if diff := difference(captured, stored); len(diff) > 0 {
		t.Errorf("screens the capture script takes are missing from docs/assets/guide: %v", diff)
	}
	if diff := difference(stored, captured); len(diff) > 0 {
		t.Errorf("docs/assets/guide holds files the capture script no longer produces: %v", diff)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve guide screenshot test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", ".."))
}

// The script names a screen in one of two places: the table of hash routes it
// walks (`["dashboard", "#/dashboard"]`) or a direct `shoot("asset-detail")`
// after it has arranged the page by hand.
var (
	routedScreen = regexp.MustCompile(`\["([a-z0-9-]+)", "#/`)
	directScreen = regexp.MustCompile(`shoot\("([a-z0-9-]+)"\)`)
	guideFigure  = regexp.MustCompile(`!\[[^\]]*\]\(assets/guide/([a-z0-9-]+)\.png\)`)
)

func capturedScreens(t *testing.T, script string) map[string]struct{} {
	t.Helper()
	raw, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("open %s: %v", script, err)
	}
	names := make(map[string]struct{})
	for _, pattern := range []*regexp.Regexp{routedScreen, directScreen} {
		for _, match := range pattern.FindAllStringSubmatch(string(raw), -1) {
			names[match[1]] = struct{}{}
		}
	}
	if len(names) == 0 {
		t.Fatalf("%s names no screens; the patterns this test reads may be stale", script)
	}
	return names
}

func referencedFigures(t *testing.T, guides ...string) map[string]struct{} {
	t.Helper()
	names := make(map[string]struct{})
	for _, guide := range guides {
		raw, err := os.ReadFile(guide)
		if err != nil {
			t.Fatalf("open %s: %v", guide, err)
		}
		for _, match := range guideFigure.FindAllStringSubmatch(string(raw), -1) {
			names[match[1]] = struct{}{}
		}
	}
	if len(names) == 0 {
		t.Fatal("the guides reference no figures under assets/guide")
	}
	return names
}

func storedScreenshots(t *testing.T, directory string) map[string]struct{} {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("list %s: %v", directory, err)
	}
	names := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".png") {
			continue
		}
		names[strings.TrimSuffix(entry.Name(), ".png")] = struct{}{}
	}
	return names
}

func difference(from, against map[string]struct{}) []string {
	missing := make([]string, 0)
	for name := range from {
		if _, ok := against[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}
