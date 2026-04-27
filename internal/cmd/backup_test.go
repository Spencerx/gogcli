package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/steipete/gogcli/internal/backup"
)

func TestBackupAccountHashStableAndOpaque(t *testing.T) {
	got := backupAccountHash("  User@Example.COM ")
	want := backupAccountHash("user@example.com")
	if got != want {
		t.Fatalf("hash not normalized: got %s want %s", got, want)
	}
	if len(got) != 24 {
		t.Fatalf("hash length = %d, want 24 hex chars", len(got))
	}
	if strings.Contains(got, "user") || strings.Contains(got, "example") {
		t.Fatalf("hash leaks account text: %s", got)
	}
}

func TestBuildGmailMessageShardsBucketsSortsAndChunks(t *testing.T) {
	accountHash := "accthash"
	messages := []gmailBackupMessage{
		{ID: "march-new", InternalDate: mustUnixMilli(t, "2026-03-02T10:00:00Z"), Raw: "raw-3"},
		{ID: "april-later", InternalDate: mustUnixMilli(t, "2026-04-02T10:00:00Z"), Raw: "raw-2"},
		{ID: "april-earlier-b", InternalDate: mustUnixMilli(t, "2026-04-01T10:00:00Z"), Raw: "raw-1b"},
		{ID: "april-earlier-a", InternalDate: mustUnixMilli(t, "2026-04-01T10:00:00Z"), Raw: "raw-1a"},
	}

	shards, err := buildGmailMessageShards(accountHash, messages, 2)
	if err != nil {
		t.Fatalf("buildGmailMessageShards: %v", err)
	}
	if len(shards) != 3 {
		t.Fatalf("len(shards) = %d, want 3", len(shards))
	}
	wantPaths := []string{
		"data/gmail/accthash/messages/2026/03/part-0001.jsonl.gz.age",
		"data/gmail/accthash/messages/2026/04/part-0001.jsonl.gz.age",
		"data/gmail/accthash/messages/2026/04/part-0002.jsonl.gz.age",
	}
	for i, want := range wantPaths {
		if shards[i].Path != want {
			t.Fatalf("shards[%d].Path = %q, want %q", i, shards[i].Path, want)
		}
	}
	if shards[0].Rows != 1 || shards[1].Rows != 2 || shards[2].Rows != 1 {
		t.Fatalf("unexpected row counts: %d %d %d", shards[0].Rows, shards[1].Rows, shards[2].Rows)
	}

	var aprilFirst []gmailBackupMessage
	if err := backup.DecodeJSONL(shards[1].Plaintext, &aprilFirst); err != nil {
		t.Fatalf("DecodeJSONL: %v", err)
	}
	gotIDs := []string{aprilFirst[0].ID, aprilFirst[1].ID}
	wantIDs := []string{"april-earlier-a", "april-earlier-b"}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("april shard IDs = %v, want %v", gotIDs, wantIDs)
		}
	}
}

func TestMergeBackupSnapshotsKeepsCountsAndShardOrder(t *testing.T) {
	left := backup.Snapshot{
		Services: []string{"gmail"},
		Accounts: []string{"acct1"},
		Counts:   map[string]int{"gmail.messages": 2},
		Shards:   []backup.PlainShard{{Path: "data/gmail/acct1/messages/2026/04/part-0001.jsonl.gz.age"}},
	}
	right := backup.Snapshot{
		Services: []string{"calendar"},
		Accounts: []string{"acct1"},
		Counts:   map[string]int{"calendar.events": 3},
		Shards:   []backup.PlainShard{{Path: "data/calendar/acct1/events.jsonl.gz.age"}},
	}

	merged := mergeBackupSnapshots(left, right)
	if merged.Counts["gmail.messages"] != 2 || merged.Counts["calendar.events"] != 3 {
		t.Fatalf("unexpected counts: %+v", merged.Counts)
	}
	if len(merged.Shards) != 2 || merged.Shards[0].Path != left.Shards[0].Path || merged.Shards[1].Path != right.Shards[0].Path {
		t.Fatalf("unexpected shard order: %+v", merged.Shards)
	}
}

func mustUnixMilli(t *testing.T, value string) int64 {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed.UnixMilli()
}
