package filemgr

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const maxRemoteDownloadBytes = 512 << 20 // 512 MiB

type RemoteDownloadResult struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

var contentDispositionFilename = regexp.MustCompile(`filename\*?=(?:UTF-8''|")?([^";]+)`)

func (s *Service) DownloadFromURL(dir, rawURL, filename string) (*RemoteDownloadResult, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("URL 不能为空")
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("无效的 URL")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("仅支持 http/https 链接")
	}
	if err := assertPublicHTTPURL(u); err != nil {
		return nil, err
	}

	destDir, err := s.resolvePath(dir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(destDir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("目标路径不是目录")
	}

	client := &http.Client{
		Timeout: 30 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("重定向次数过多")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("不允许的重定向协议")
			}
			if err := assertPublicHTTPURL(req.URL); err != nil {
				return err
			}
			return nil
		},
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "OWPanel-FileManager/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	name := strings.TrimSpace(filename)
	if name == "" {
		name = filenameFromHTTP(u, resp)
	}
	name = filepath.Base(filepath.FromSlash(name))
	if name == "" || name == "." || name == ".." {
		name = "download"
	}

	target := filepath.Join(destDir, name)
	out, err := os.Create(target)
	if err != nil {
		return nil, err
	}

	limited := io.LimitReader(resp.Body, maxRemoteDownloadBytes+1)
	written, err := io.Copy(out, limited)
	closeErr := out.Close()
	if err != nil {
		_ = os.Remove(target)
		return nil, fmt.Errorf("写入文件失败: %w", err)
	}
	if closeErr != nil {
		_ = os.Remove(target)
		return nil, closeErr
	}
	if written > maxRemoteDownloadBytes {
		_ = os.Remove(target)
		return nil, fmt.Errorf("文件超过最大限制 %d MB", maxRemoteDownloadBytes>>20)
	}

	return &RemoteDownloadResult{
		Path: target,
		Name: name,
		Size: written,
	}, nil
}

func assertPublicHTTPURL(u *url.URL) error {
	if u == nil || u.Hostname() == "" {
		return fmt.Errorf("无效的 URL")
	}
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return fmt.Errorf("不允许下载内网地址")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("无法解析主机: %w", err)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("不允许下载内网地址")
		}
		// Cloud metadata / CGNAT
		if ip4 := ip.To4(); ip4 != nil {
			if ip4[0] == 169 && ip4[1] == 254 {
				return fmt.Errorf("不允许下载内网地址")
			}
		}
	}
	return nil
}

func filenameFromHTTP(u *url.URL, resp *http.Response) string {
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if m := contentDispositionFilename.FindStringSubmatch(cd); len(m) > 1 {
			name := strings.Trim(strings.TrimSpace(m[1]), `"`)
			if decoded, err := url.PathUnescape(name); err == nil {
				name = decoded
			}
			if name != "" {
				return name
			}
		}
	}
	base := filepath.Base(u.Path)
	if base != "" && base != "/" && base != "." {
		return base
	}
	return "download"
}
