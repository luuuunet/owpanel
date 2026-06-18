package mail

import (
	"os/exec"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns a Dovecot-compatible password hash for passwd-file auth.
func HashPassword(plain string) (string, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return "", nil
	}
	if _, err := exec.LookPath("doveadm"); err == nil {
		out, err := exec.Command("doveadm", "pw", "-s", "SHA512-CRYPT", "-p", plain).CombinedOutput()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return "{BLF-CRYPT}" + string(hash), nil
}

// IsHashed reports whether a stored password already looks like a Dovecot hash.
func IsHashed(stored string) bool {
	stored = strings.TrimSpace(stored)
	return strings.HasPrefix(stored, "{") && strings.Contains(stored, "}")
}
