package platform

import "testing"

func TestDetectLinuxFamily(t *testing.T) {
	info := Detect()
	if info.GOOS == "" {
		t.Fatal("expected GOOS")
	}
	if info.GOOS == "linux" && info.PackageManager == "" {
		t.Log("linux host without apt/dnf/yum (CI?)")
	}
}
