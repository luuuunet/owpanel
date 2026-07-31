package filemgr

import (
	"path/filepath"
	"strings"
)

// common binary / non-text extensions that must not open in the code editor
var binaryExts = map[string]struct{}{
	".exe": {}, ".dll": {}, ".so": {}, ".dylib": {}, ".bin": {}, ".o": {}, ".a": {}, ".lib": {}, ".obj": {},
	".apk": {}, ".aab": {}, ".ipa": {}, ".msi": {}, ".dmg": {}, ".iso": {}, ".img": {}, ".deb": {}, ".rpm": {},
	".jar": {}, ".war": {}, ".ear": {}, ".class": {}, ".pyc": {}, ".pyo": {}, ".wasm": {}, ".pdb": {},
	".zip": {}, ".rar": {}, ".7z": {}, ".tar": {}, ".gz": {}, ".tgz": {}, ".bz2": {}, ".xz": {}, ".zst": {},
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".webp": {}, ".bmp": {}, ".ico": {}, ".tif": {}, ".tiff": {},
	".mp3": {}, ".mp4": {}, ".avi": {}, ".mov": {}, ".mkv": {}, ".webm": {}, ".flac": {}, ".wav": {}, ".ogg": {},
	".pdf": {}, ".doc": {}, ".docx": {}, ".xls": {}, ".xlsx": {}, ".ppt": {}, ".pptx": {},
	".woff": {}, ".woff2": {}, ".ttf": {}, ".otf": {}, ".eot": {},
	".sqlite": {}, ".db": {}, ".dat": {}, ".pak": {}, ".crx": {},
}

func IsBinaryFilename(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return false
	}
	_, ok := binaryExts[ext]
	return ok
}

// LooksLikeBinary reports whether content appears non-textual (NUL bytes in sample).
func LooksLikeBinary(data []byte) bool {
	n := len(data)
	if n > 8192 {
		n = 8192
	}
	for i := 0; i < n; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}
