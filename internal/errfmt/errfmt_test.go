package errfmt

import (
	"errors"
	"strings"
	"testing"

	"github.com/99designs/keyring"
	ggoogleapi "google.golang.org/api/googleapi"

	"github.com/steipete/gogcli/internal/config"
	gogapi "github.com/steipete/gogcli/internal/googleapi"
)

func TestFormat_Nil(t *testing.T) {
	if got := Format(nil); got != "" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestFormat_AuthRequired(t *testing.T) {
	err := &gogapi.AuthRequiredError{Service: "gmail", Email: "a@b.com", Cause: keyring.ErrKeyNotFound}
	got := Format(err)
	if got == "" {
		t.Fatalf("expected message")
	}
	if !containsAll(got, "gog auth add", "a@b.com", "gmail") {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestFormat_CredentialsMissing(t *testing.T) {
	err := &config.CredentialsMissingError{Path: "/tmp/creds.json", Cause: errors.New("nope")}
	got := Format(err)
	if !containsAll(got, "gog auth credentials", "/tmp/creds.json") {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestFormat_KeyNotFound(t *testing.T) {
	got := Format(keyring.ErrKeyNotFound)
	if !containsAll(got, "Secret not found", "gog auth add") {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestFormat_GoogleAPIError(t *testing.T) {
	err := &ggoogleapi.Error{
		Code:    403,
		Message: "nope",
		Errors: []ggoogleapi.ErrorItem{
			{Reason: "insufficientPermissions"},
		},
	}
	got := Format(err)
	if !containsAll(got, "403", "insufficientPermissions", "nope") {
		t.Fatalf("unexpected: %q", got)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
