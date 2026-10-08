package auth

import (
	"strings"
	"testing"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := HashPassword("CorrectHorse!42")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("HashPassword() = %q, want Argon2id prefix", hash)
	}
	valid, err := VerifyPassword("CorrectHorse!42", hash)
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !valid {
		t.Fatal("VerifyPassword() rejected the correct password")
	}
	valid, err = VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword(wrong) error = %v", err)
	}
	if valid {
		t.Fatal("VerifyPassword() accepted a wrong password")
	}
}

func TestDefaultPasswordPolicy(t *testing.T) {
	policy := DefaultPasswordPolicy()
	if err := policy.Validate("CorrectHorse!42"); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	for _, password := range []string{
		"Short!2A",
		"alllowercase!42",
		"ALLUPPERCASE!42",
		"NoNumbersHere!",
		"NoSymbolsHere42",
	} {
		if err := policy.Validate(password); err == nil {
			t.Errorf("Validate(%q) accepted a policy violation", password)
		}
	}
}
