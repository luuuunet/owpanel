package filemgr

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ArchiveResult struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func (s *Service) Compress(paths []string, format, dest string) (*ArchiveResult, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("no paths selected")
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "zip"
	}
	resolved := make([]string, 0, len(paths))
	for _, p := range paths {
		rp, err := s.resolvePath(p)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, rp)
	}
	destPath, err := s.resolvePath(dest)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return nil, err
	}
	switch format {
	case "zip":
		err = compressZip(resolved, destPath)
	case "tar.gz", "tgz":
		err = compressTarGz(resolved, destPath)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(destPath)
	if err != nil {
		return nil, err
	}
	return &ArchiveResult{Path: destPath, Size: st.Size()}, nil
}

func (s *Service) Extract(archivePath, destDir string) error {
	src, err := s.resolvePath(archivePath)
	if err != nil {
		return err
	}
	dest, err := s.resolvePath(destDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	lower := strings.ToLower(src)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(src, dest)
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(src, dest)
	default:
		return fmt.Errorf("unsupported archive: %s", filepath.Base(src))
	}
}

func compressZip(paths []string, dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			return err
		}
		base := filepath.Base(root)
		if info.IsDir() {
			err = filepath.Walk(root, func(path string, fi os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				name := filepath.Join(base, rel)
				name = filepath.ToSlash(name)
				if fi.IsDir() {
					if rel == "." {
						return nil
					}
					_, err = zw.Create(name + "/")
					return err
				}
				w, err := zw.Create(name)
				if err != nil {
					return err
				}
				rf, err := os.Open(path)
				if err != nil {
					return err
				}
				defer rf.Close()
				_, err = io.Copy(w, rf)
				return err
			})
			if err != nil {
				return err
			}
		} else {
			w, err := zw.Create(base)
			if err != nil {
				return err
			}
			rf, err := os.Open(root)
			if err != nil {
				return err
			}
			defer rf.Close()
			if _, err := io.Copy(w, rf); err != nil {
				return err
			}
		}
	}
	return zw.Close()
}

func compressTarGz(paths []string, dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			return err
		}
		base := filepath.Base(root)
		if info.IsDir() {
			err = filepath.Walk(root, func(path string, fi os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				name := filepath.Join(base, rel)
				name = filepath.ToSlash(name)
				hdr, err := tar.FileInfoHeader(fi, "")
				if err != nil {
					return err
				}
				hdr.Name = name
				if err := tw.WriteHeader(hdr); err != nil {
					return err
				}
				if fi.IsDir() {
					return nil
				}
				rf, err := os.Open(path)
				if err != nil {
					return err
				}
				defer rf.Close()
				_, err = io.Copy(tw, rf)
				return err
			})
			if err != nil {
				return err
			}
		} else {
			hdr, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			hdr.Name = base
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			rf, err := os.Open(root)
			if err != nil {
				return err
			}
			defer rf.Close()
			if _, err := io.Copy(tw, rf); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	dest = filepath.Clean(dest)
	for _, f := range r.File {
		target := filepath.Join(dest, f.Name)
		target = filepath.Clean(target)
		if !strings.HasPrefix(target, dest+string(os.PathSeparator)) && target != dest {
			return fmt.Errorf("invalid zip entry: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	dest = filepath.Clean(dest)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dest, hdr.Name)
		target = filepath.Clean(target)
		if !strings.HasPrefix(target, dest+string(os.PathSeparator)) && target != dest {
			return fmt.Errorf("invalid tar entry: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		default:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}
