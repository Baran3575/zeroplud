package tui

import (
	"strings"
	"testing"
)

func TestDetectLeakedControlTokensDetectsControlToken(t *testing.T) {
	warning := detectLeakedControlTokens("Hello <|im_end|> world")
	if warning == "" {
		t.Fatal("expected warning for leaked control token")
	}
	if !strings.Contains(warning, "<|im_end|>") {
		t.Fatalf("expected warning to mention the leaked token, got %q", warning)
	}
}

func TestDetectLeakedControlTokensDetectsKnownLeakedTokens(t *testing.T) {
	cases := []string{
		"<|eom|>",
		"<|im_end|>",
		"<|endoftext|>",
		"<|end|>",
		"<|startoftext|>",
		"<|tool_call|>",
		"<|tool_result|>",
		"</tool>",
	}
	for _, tc := range cases {
		if warning := detectLeakedControlTokens(tc); warning == "" {
			t.Fatalf("expected warning for %q", tc)
		}
	}
}

func TestDetectLeakedControlTokensPassesCleanText(t *testing.T) {
	cases := []string{
		"",
		"Hello world",
		"Normal text with <b>HTML tags</b>",
		"Markdown with <angle brackets>",
		"Pipes | in text",
	}
	for _, tc := range cases {
		if warning := detectLeakedControlTokens(tc); warning != "" {
			t.Fatalf("unexpected warning for %q: %q", tc, warning)
		}
	}
}

func TestReduceTranscriptAppendsControlTokenWarning(t *testing.T) {
	rows := reduceTranscript([]transcriptRow{}, transcriptAction{
		kind: actionAppendAssistant,
		text: "Here is the answer <|im_end|>",
	})
	found := false
	for _, row := range rows {
		if row.kind == rowSystem && strings.Contains(row.text, "Control token leaked:") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected a system warning row for control token leakage")
	}
}

func TestReduceTranscriptSkipsWarningForCleanText(t *testing.T) {
	rows := reduceTranscript([]transcriptRow{}, transcriptAction{
		kind: actionAppendAssistant,
		text: "Clean text without control tokens",
	})
	for _, row := range rows {
		if row.kind == rowSystem && strings.Contains(row.text, "Control token leaked:") {
			t.Fatalf("unexpected control token warning for clean text")
		}
	}
}

// appendTranscriptRowsDedup must produce byte-identical output to repeated
// appendTranscriptRow (the O(n²) form it replaces), including keyed-row dedup and
// always-append for unkeyed rows.
func TestAppendTranscriptRowsDedupMatchesPerRow(t *testing.T) {
	newRows := []transcriptRow{
		{kind: rowToolCall, runID: 1, id: "a"},
		{kind: rowSystem, text: "note"},
		{kind: rowToolCall, runID: 1, id: "a"}, // duplicate keyed row -> skipped
		{kind: rowToolResult, runID: 1, id: "b"},
		{kind: rowSystem, text: "note"},        // unkeyed duplicate -> still appended
		{kind: rowToolCall, runID: 2, id: "a"}, // same id, different run -> distinct key
	}
	base := initialTranscript()

	want := append([]transcriptRow{}, base...)
	for _, r := range newRows {
		want = appendTranscriptRow(want, r)
	}
	got := appendTranscriptRowsDedup(append([]transcriptRow{}, base...), newRows)

	if len(got) != len(want) {
		t.Fatalf("length mismatch: bulk=%d per-row=%d", len(got), len(want))
	}
	for i := range want {
		if got[i].kind != want[i].kind || got[i].id != want[i].id || got[i].runID != want[i].runID || got[i].text != want[i].text {
			t.Fatalf("row %d differs:\n bulk=%+v\n per =%+v", i, got[i], want[i])
		}
	}
}
