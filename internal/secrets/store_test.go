package secrets

import (
	"testing"
	"time"

	"github.com/99designs/keyring"
)

func TestTokenKey(t *testing.T) {
	if got := tokenKey("a@b.com"); got != "token:a@b.com" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestParseTokenKey(t *testing.T) {
	email, ok := ParseTokenKey("token:a@b.com")
	if !ok {
		t.Fatalf("expected ok")
	}
	if email != "a@b.com" {
		t.Fatalf("unexpected: %q", email)
	}
	if _, ok := ParseTokenKey("nope"); ok {
		t.Fatalf("expected not ok")
	}
}

func TestKeyringStore_TokenRoundtrip(t *testing.T) {
	s := &KeyringStore{ring: keyring.NewArrayKeyring(nil)}

	createdAt := time.Date(2025, 12, 12, 0, 0, 0, 0, time.UTC)
	if err := s.SetToken("A@B.COM", Token{
		Email:        "A@B.COM",
		Services:     []string{"gmail"},
		Scopes:       []string{"scope1"},
		CreatedAt:    createdAt,
		RefreshToken: "rt",
	}); err != nil {
		t.Fatalf("SetToken: %v", err)
	}

	got, err := s.GetToken("a@b.com")
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if got.Email != "a@b.com" {
		t.Fatalf("email: %q", got.Email)
	}
	if got.RefreshToken != "rt" {
		t.Fatalf("refresh token mismatch")
	}
	if got.CreatedAt.IsZero() {
		t.Fatalf("expected createdAt")
	}

	list, err := s.ListTokens()
	if err != nil {
		t.Fatalf("ListTokens: %v", err)
	}
	if len(list) != 1 || list[0].Email != "a@b.com" {
		t.Fatalf("unexpected list: %#v", list)
	}

	if err := s.DeleteToken("a@b.com"); err != nil {
		t.Fatalf("DeleteToken: %v", err)
	}
	if _, err := s.GetToken("a@b.com"); err == nil {
		t.Fatalf("expected error after delete")
	}
}
