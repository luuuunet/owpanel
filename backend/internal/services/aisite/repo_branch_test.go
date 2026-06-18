package aisite

import "testing"

func TestResolveGitBranchLaravelDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("network")
	}
	got := resolveGitBranch("https://github.com/laravel/laravel", "main", "")
	if got != "13.x" {
		t.Fatalf("resolveGitBranch(laravel/laravel, main) = %q, want 13.x", got)
	}
}
