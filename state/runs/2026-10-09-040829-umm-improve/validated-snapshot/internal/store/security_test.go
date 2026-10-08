package store

import (
	"slices"
	"testing"
)

func TestLoginAccountIdentityCanBeClearedWithoutTheAddress(t *testing.T) {
	identities := LoginIdentities("  Alice  ", "203.0.113.8")
	if !slices.Equal(identities, []string{"ip:203.0.113.8", "user:alice"}) {
		t.Fatalf("unexpected login identities: %#v", identities)
	}
	if account := LoginAccountIdentity("  Alice  "); account != "user:alice" {
		t.Fatalf("unexpected account identity: %q", account)
	}
}
