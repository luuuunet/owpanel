package database

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type mysqlServerInfo struct {
	Engine  string // mysql | mariadb
	Version string // e.g. 5.7.44
	Major   int
	Minor   int
	Patch   int
	Raw     string
}

var mysqlVersionNumRe = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

func (info mysqlServerInfo) useCachingSha2() bool {
	if info.Engine == "mariadb" {
		return false
	}
	return info.Major >= 8
}

func (info mysqlServerInfo) supportsCreateUserIfNotExists() bool {
	if info.Engine == "mariadb" {
		return info.Major > 10 || (info.Major == 10 && info.Minor >= 1)
	}
	if info.Major > 5 {
		return true
	}
	return info.Major == 5 && info.Minor >= 7
}

func (info mysqlServerInfo) identifiedBy(password string) string {
	esc := mysqlEscapeSQL(password)
	if info.useCachingSha2() {
		return fmt.Sprintf("IDENTIFIED WITH caching_sha2_password BY '%s'", esc)
	}
	return fmt.Sprintf("IDENTIFIED BY '%s'", esc)
}

func parseMySQLServerVersion(raw string) mysqlServerInfo {
	info := mysqlServerInfo{Engine: "mysql", Raw: strings.TrimSpace(raw)}
	if raw == "" {
		return info
	}
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "mariadb") {
		info.Engine = "mariadb"
	}
	m := mysqlVersionNumRe.FindStringSubmatch(raw)
	if len(m) >= 4 {
		info.Major, _ = strconv.Atoi(m[1])
		info.Minor, _ = strconv.Atoi(m[2])
		info.Patch, _ = strconv.Atoi(m[3])
		info.Version = fmt.Sprintf("%d.%d.%d", info.Major, info.Minor, info.Patch)
	}
	return info
}

func (s *Service) queryMySQLServer() mysqlServerInfo {
	out, err := s.mysqlRootExec("-N", "-e", "SELECT VERSION();")
	if err == nil {
		if info := parseMySQLServerVersion(string(out)); info.Version != "" {
			return info
		}
	}
	bin, err := findBinary("mysql", "mariadb")
	if err != nil {
		return mysqlServerInfo{Engine: "mysql"}
	}
	clientOut, err := exec.Command(bin, "--version").CombinedOutput()
	if err != nil {
		return mysqlServerInfo{Engine: detectMySQLEngine()}
	}
	engine, display := parseMySQLVersionOutput(strings.TrimSpace(string(clientOut)))
	info := parseMySQLServerVersion(display)
	if info.Version == "" {
		info.Version = display
	}
	info.Engine = engine
	if info.Engine == "" {
		info.Engine = "mysql"
	}
	return info
}

func mysqlEnsureUserSQL(info mysqlServerInfo, user, host, password string) string {
	escUser := mysqlEscapeSQL(user)
	escHost := mysqlEscapeSQL(host)
	id := info.identifiedBy(password)
	if info.supportsCreateUserIfNotExists() {
		return fmt.Sprintf(
			"CREATE USER IF NOT EXISTS '%s'@'%s' %s; ALTER USER '%s'@'%s' %s;",
			escUser, escHost, id, escUser, escHost, id,
		)
	}
	return fmt.Sprintf(
		"GRANT USAGE ON *.* TO '%s'@'%s' %s; ALTER USER '%s'@'%s' %s;",
		escUser, escHost, id, escUser, escHost, id,
	)
}

func mysqldumpExtraArgs(dumpBin string) []string {
	out, err := exec.Command(dumpBin, "--version").CombinedOutput()
	if err != nil {
		return nil
	}
	lower := strings.ToLower(string(out))
	if strings.Contains(lower, "mysqldump") && strings.Contains(lower, "ver 8") {
		return []string{"--column-statistics=0"}
	}
	return nil
}
