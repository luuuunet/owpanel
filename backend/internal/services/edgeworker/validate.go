package edgeworker

import (
	"fmt"
	"regexp"
	"strings"
)

var luaBlocked = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bos\.execute\b`),
	regexp.MustCompile(`(?i)\bios?\.popen\b`),
	regexp.MustCompile(`(?i)\bio\.open\b`),
	regexp.MustCompile(`(?i)\bio\.input\b`),
	regexp.MustCompile(`(?i)\bio\.output\b`),
	regexp.MustCompile(`(?i)\bio\.tmpfile\b`),
	regexp.MustCompile(`(?i)\bloadfile\b`),
	regexp.MustCompile(`(?i)\bdofile\b`),
	regexp.MustCompile(`(?i)\brequire\s*\(\s*["']socket`),
	regexp.MustCompile(`(?i)\brequire\s*\(\s*["']luasocket`),
	regexp.MustCompile(`(?i)\bngx\.exec\s*\(`),
	regexp.MustCompile(`(?i)\bos\.remove\b`),
	regexp.MustCompile(`(?i)\bos\.rename\b`),
}

var njsBlocked = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\brequire\s*\(\s*['"]child_process`),
	regexp.MustCompile(`(?i)\brequire\s*\(\s*['"]fs['"]`),
	regexp.MustCompile(`(?i)\brequire\s*\(\s*['"]net['"]`),
	regexp.MustCompile(`(?i)\brequire\s*\(\s*['"]http['"]`),
	regexp.MustCompile(`(?i)\bprocess\.`),
	regexp.MustCompile(`(?i)\beval\s*\(`),
	regexp.MustCompile(`(?i)\bFunction\s*\(`),
}

func ValidateScript(scriptType, script string) error {
	script = strings.TrimSpace(script)
	if script == "" {
		return fmt.Errorf("script body is required")
	}
	switch scriptType {
	case "lua":
		for _, re := range luaBlocked {
			if re.MatchString(script) {
				return fmt.Errorf("script contains blocked pattern: %s", re.String())
			}
		}
	case "njs":
		for _, re := range njsBlocked {
			if re.MatchString(script) {
				return fmt.Errorf("script contains blocked pattern: %s", re.String())
			}
		}
	case "template":
		if strings.Contains(script, "..") {
			return fmt.Errorf("template cannot contain path traversal")
		}
	default:
		return fmt.Errorf("unsupported script_type: %s", scriptType)
	}
	return nil
}
