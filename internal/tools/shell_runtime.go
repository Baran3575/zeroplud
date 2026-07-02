package tools

import (
	"regexp"
	"strings"
)

type ShellRuntime interface {
	GOOS() string
	Executable() string
	Syntax() string
	PreFlightCheck(command string) *shellIssue
	PostFlightCheck(command string, output string) *shellIssue
	Guidance() string
}

type shellIssue struct {
	Kind       string
	Message    string
	Suggestion string
}

var (
	windowsBashStyleCDPattern = regexp.MustCompile(`(?i)(^|[&|;]\s*)cd\s+/(?:[a-ce-z0-9_./~-]|d[a-z0-9_./~-])[a-z0-9_./~-]*`)
	windowsLSCommandPattern   = regexp.MustCompile(`(?i)(^|[&|;]\s*)ls\b(?:\s+|$)`)
	windowsPosixUtilityPattern = regexp.MustCompile(`(?i)(^|[&|;]\s*)(head|tail|grep|wc|awk|sed|cut|xargs|tr)\b`)
	windowsInlineCodeRe       = regexp.MustCompile(`(?i)(?:^|[|;&]\s*)(?:python|python3|py|node)\s+(?:-c|-e)\s+(["'])`)
)

type windowsCmdRuntime struct{}

func (windowsCmdRuntime) GOOS() string       { return "windows" }
func (windowsCmdRuntime) Executable() string { return "cmd.exe" }
func (windowsCmdRuntime) Syntax() string     { return "Windows cmd.exe" }
func (windowsCmdRuntime) Guidance() string {
	return "Uses Windows cmd.exe syntax on Windows; prefer cwd over cd when changing directories."
}

func (windowsCmdRuntime) PreFlightCheck(command string) *shellIssue {
	trimmed := strings.TrimSpace(command)
	if detectWindowsInlineCodeQuoting(trimmed) {
		return &shellIssue{
			Kind:       "windows_shell_quoting",
			Message:    "Command uses -c/-e inline code with nested quotes that Windows cmd.exe will mangle.",
			Suggestion: "Write the code to a temp .py/.js file with write_file, then run that file with the interpreter.",
		}
	}
	if windowsBashStyleCDPattern.MatchString(trimmed) ||
		windowsLSCommandPattern.MatchString(trimmed) {
		return &shellIssue{
			Kind:       "windows_shell_syntax",
			Message:    "Command looks like POSIX/Bash syntax, but Zero runs bash tool commands through Windows cmd.exe on this host.",
			Suggestion: "Use the cwd argument instead of cd, use Windows cmd.exe syntax, or use native tools such as list_directory, read_file, grep, and glob.",
		}
	}
	if windowsPosixUtilityPattern.MatchString(trimmed) {
		return &shellIssue{
			Kind:       "windows_shell_syntax",
			Message:    "Command uses a POSIX utility (head/tail/grep/wc/awk/sed/cut/xargs/tr) that Windows cmd.exe does not have.",
			Suggestion: "Use cmd.exe equivalents (e.g. `findstr` for grep, `more` to page output) or Zero's native tools (grep, read_file with offset/limit) instead of piping to a POSIX utility.",
		}
	}
	return nil
}

func (windowsCmdRuntime) PostFlightCheck(command string, output string) *shellIssue {
	haystack := strings.ToLower(command + "\n" + output)
	if strings.Contains(haystack, "the syntax of the command is incorrect") ||
		strings.Contains(haystack, "is not recognized as an internal or external command") {
		return &shellIssue{
			Kind:       "windows_shell_syntax",
			Message:    "Windows cmd.exe rejected the command syntax.",
			Suggestion: "Translate the command to Windows cmd.exe syntax, set the bash tool cwd argument instead of running cd, or prefer native Zero tools for file inspection.",
		}
	}
	return nil
}

type posixShellRuntime struct {
	goos string
}

func (r posixShellRuntime) GOOS() string       { return r.goos }
func (r posixShellRuntime) Executable() string { return "/bin/sh" }
func (r posixShellRuntime) Syntax() string     { return "/bin/sh" }

func (posixShellRuntime) PreFlightCheck(string) *shellIssue               { return nil }
func (posixShellRuntime) PostFlightCheck(string, string) *shellIssue      { return nil }

func (r posixShellRuntime) Guidance() string {
	guidance := "Uses " + r.Syntax() + " syntax."
	if r.goos == "darwin" {
		guidance += " To find or stop a process, use `lsof -i :PORT` (or `lsof -nP -iTCP -sTCP:LISTEN`) for the PID then `kill <pid>`; `ps` and `pgrep` do not work under the sandbox."
	}
	return guidance
}

func getShellRuntime(goos string) ShellRuntime {
	if goos == "windows" {
		return windowsCmdRuntime{}
	}
	return posixShellRuntime{goos: goos}
}

// detectWindowsInlineCodeQuoting checks whether command uses -c / -e with
// inline code whose quoted payload contains additional quote characters that
// cmd.exe will mangle.
func detectWindowsInlineCodeQuoting(command string) bool {
	matches := windowsInlineCodeRe.FindStringSubmatch(command)
	if len(matches) < 2 {
		return false
	}
	quote := matches[1]
	idx := strings.Index(command, matches[0])
	if idx < 0 {
		return false
	}
	payload := command[idx+len(matches[0]):]
	endQuote := strings.IndexByte(payload, quote[0])
	if endQuote < 0 {
		return false
	}
	inner := payload[:endQuote]
	otherQuote := `'`
	if quote == "'" {
		otherQuote = `"`
	}
	return strings.Contains(inner, otherQuote)
}

func shellGuidanceForGOOS(goos string) string {
	return getShellRuntime(goos).Guidance()
}

func detectShellCommandIssue(command string, goos string) *shellIssue {
	return getShellRuntime(goos).PreFlightCheck(command)
}

func detectShellOutputIssue(command string, output string, goos string) *shellIssue {
	return getShellRuntime(goos).PostFlightCheck(command, output)
}

func appendShellIssueHint(output string, issue shellIssue) string {
	output = strings.TrimRight(output, "\r\n")
	hint := "[zero] shell issue: " + issue.Message
	if strings.TrimSpace(issue.Suggestion) != "" {
		hint += "\nSuggestion: " + issue.Suggestion
	}
	if output == "" {
		return hint
	}
	return output + "\n" + hint
}
