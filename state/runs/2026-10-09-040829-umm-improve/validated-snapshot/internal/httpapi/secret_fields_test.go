package httpapi

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/hkjang/umm/internal/store"
)

// looksLikeASecret is deliberately about the name rather than the value.
//
// A field called something_key holds a credential whatever happens to be in it
// at the moment the test runs, and an empty one on a fresh install must not be
// the reason this passes.
func looksLikeASecret(field string) bool {
	for _, marker := range []string{"_key", "_secret", "_token", "_password", "_credential"} {
		if strings.HasSuffix(field, marker) {
			return true
		}
	}
	return false
}

// Every credential in the settings must be masked when read and encrypted when
// written, and secretFields is the one list that decides both.
//
// Adding a second secret to a section is how this goes wrong: the field is added
// to the struct and to the form, the list is forgotten, and the settings API
// starts returning a key in plain text to anyone who can read settings. umm came
// within one edit of exactly that when embeddings gained their own key.
func TestEverySettingThatLooksLikeACredentialIsMaskedIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	sections, err := db.AllSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) == 0 {
		t.Fatal("no settings were stored, so this guard would pass without checking anything")
	}

	checked := 0
	for section, raw := range sections {
		var value map[string]any
		if json.Unmarshal(raw, &value) != nil {
			continue
		}
		masked := secretFields(section)
		for field := range value {
			if !looksLikeASecret(field) {
				continue
			}
			checked++
			if !slices.Contains(masked, field) {
				t.Errorf("%s.%s looks like a credential but is not in secretFields(%q): "+
					"it would be returned in plain text by the settings API and stored unencrypted",
					section, field, section)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no credential-shaped settings were found; the guard is not looking at anything")
	}

	// There is deliberately no check that every name in the list exists in the
	// stored settings. A field the section has never saved is simply absent —
	// embedding_api_key is, on any install that has not set one — so absence
	// cannot be told from a typo here. And a typo would fail the check above
	// anyway, because the real field would then be unmasked.
	_ = store.AllowedSetting
}
