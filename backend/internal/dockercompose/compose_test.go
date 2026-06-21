package dockercompose

import (
	"testing"
)

func TestDefaultComposeInDir(t *testing.T) {
	dir := "/opt/compose/portainer-ce"
	cases := []struct {
		file string
		want bool
	}{
		{"/opt/compose/portainer-ce/docker-compose.yml", true},
		{"/opt/compose/portainer-ce/compose.yml", true},
		{"/other/docker-compose.yml", false},
		{"", true},
	}
	for _, c := range cases {
		if got := defaultComposeInDir(dir, c.file); got != c.want {
			t.Fatalf("defaultComposeInDir(%q, %q) = %v want %v", dir, c.file, got, c.want)
		}
	}
}

func TestComposeArgsSkipsFForDefaultFile(t *testing.T) {
	dir := "/opt/compose/app"
	args := composeArgs(dir, dir+"/docker-compose.yml", "up", "-d")
	if len(args) != 2 || args[0] != "up" {
		t.Fatalf("expected up -d without -f, got %v", args)
	}
	args = composeArgs(dir, "/other/extra.yml", "up", "-d")
	if len(args) != 4 || args[0] != "-f" {
		t.Fatalf("expected -f for non-default file, got %v", args)
	}
}

func TestIsComposeMissing(t *testing.T) {
	msg := "unknown shorthand flag: 'f' in -f"
	if !isComposeMissing(fmtError(msg), msg) {
		t.Fatal("expected compose missing for docker -f error")
	}
}

func fmtError(s string) error {
	return &testError{s}
}

type testError struct{ s string }

func (e *testError) Error() string { return e.s }
