package mail

import (
	"fmt"
	"net/mail"
	"strings"
)

// sanitizeHeaderValue strips CR/LF to prevent SMTP header injection.
func sanitizeHeaderValue(v string) string {
	v = strings.ReplaceAll(v, "\r", "")
	v = strings.ReplaceAll(v, "\n", "")
	return strings.TrimSpace(v)
}

func validateMailAddress(addr string) error {
	addr = sanitizeHeaderValue(addr)
	if addr == "" {
		return fmt.Errorf("邮件地址为空")
	}
	if _, err := mail.ParseAddress(addr); err != nil {
		return fmt.Errorf("无效邮件地址: %s", addr)
	}
	return nil
}
