package tui

import "strings"

// knownControlTokens lists tokens that models may leak.
var knownControlTokens = []string{
	"<|eom|>",
	"<|im_end|>",
	"<|endoftext|>",
	"<|end|>",
	"<|startoftext|>",
	"<\\s>",
	"<|tool_call|>",
	"<|tool_result|>",
	"</tool>",
}

// DetectControlTokenLeakage scans text for leaked control tokens.
// Returns the first leaked token found, or empty string if none.
func DetectControlTokenLeakage(text string) string {
	lower := strings.ToLower(text)
	for _, tok := range knownControlTokens {
		if strings.Contains(lower, strings.ToLower(tok)) {
			return tok
		}
	}
	return ""
}
