package database

import "strings"

const (
	AccessModeLocal  = "local"
	AccessModeRemote = "remote"
	AccessModeBoth   = "both"
)

func NormalizeAccessMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case AccessModeRemote, "everyone":
		return AccessModeRemote
	case AccessModeBoth, "all":
		return AccessModeBoth
	default:
		return AccessModeLocal
	}
}

func AccessModeFromInstance(stored string, allowRemote bool) string {
	if strings.TrimSpace(stored) != "" {
		return NormalizeAccessMode(stored)
	}
	if allowRemote {
		return AccessModeBoth
	}
	return AccessModeLocal
}

func AllowRemoteFromAccessMode(mode string) bool {
	m := NormalizeAccessMode(mode)
	return m == AccessModeRemote || m == AccessModeBoth
}

func mysqlHostsForAccessMode(mode string) []string {
	switch NormalizeAccessMode(mode) {
	case AccessModeRemote:
		return []string{"%"}
	case AccessModeBoth:
		return []string{"localhost", "127.0.0.1", "%"}
	default:
		return []string{"localhost", "127.0.0.1"}
	}
}
