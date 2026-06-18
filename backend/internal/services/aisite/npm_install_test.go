package aisite

import (
	"strings"
	"testing"
)

func TestNpmInstallBlockUsesLockfileGuard(t *testing.T) {
	block := npmInstallBlock()
	for _, want := range []string{
		"pnpm-lock.yaml",
		"package-lock.json",
		"catalog:",
		"corepack enable",
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("npmInstallBlock missing %q", want)
		}
	}
	if strings.Contains(block, "npm ci || npm install") {
		t.Fatal("should not use blind npm ci fallback")
	}
}
