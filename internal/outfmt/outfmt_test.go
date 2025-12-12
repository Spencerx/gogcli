package outfmt

import (
	"bytes"
	"context"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in      string
		want    Mode
		wantErr bool
	}{
		{"", ModeText, false},
		{"text", ModeText, false},
		{"json", ModeJSON, false},
		{" JSON ", ModeJSON, false},
		{"nope", "", true},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in)
		if tt.wantErr && err == nil {
			t.Fatalf("Parse(%q): expected error", tt.in)
		}
		if !tt.wantErr && err != nil {
			t.Fatalf("Parse(%q): %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("Parse(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestContextMode(t *testing.T) {
	ctx := context.Background()
	if FromContext(ctx) != ModeText {
		t.Fatalf("expected default text")
	}
	ctx = WithMode(ctx, ModeJSON)
	if !IsJSON(ctx) {
		t.Fatalf("expected json")
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, map[string]any{"ok": true}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatalf("expected output")
	}
}
