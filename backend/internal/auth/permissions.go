package auth

import (
	"encoding/json"
	"strings"
)

type UserPermissions struct {
	Websites  bool `json:"websites"`
	Databases bool `json:"databases"`
	Files     bool `json:"files"`
	Docker    bool `json:"docker"`
	FTP       bool `json:"ftp"`
	Mail      bool `json:"mail"`
	Backup    bool `json:"backup"`
	Monitor   bool `json:"monitor"`
	Bastion   bool `json:"bastion"`
}

func DefaultPermissions() UserPermissions {
	return UserPermissions{
		Websites: true, Databases: true, Files: true, Docker: true,
		FTP: true, Mail: true, Backup: true, Monitor: true,
	}
}

func ParsePermissions(raw string) UserPermissions {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return UserPermissions{}
	}
	var p UserPermissions
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return UserPermissions{}
	}
	return p
}

func (p UserPermissions) JSON() string {
	b, _ := json.Marshal(p)
	return string(b)
}

func (p UserPermissions) Has(perm string) bool {
	switch perm {
	case "websites":
		return p.Websites
	case "databases":
		return p.Databases
	case "files":
		return p.Files
	case "docker":
		return p.Docker
	case "ftp":
		return p.FTP
	case "mail":
		return p.Mail
	case "backup":
		return p.Backup
	case "monitor":
		return p.Monitor
	case "bastion":
		return p.Bastion
	case "settings", "users":
		return false
	default:
		return false
	}
}

func CanAccess(role, permissionsJSON, perm string) bool {
	if role == "admin" {
		return true
	}
	if role == "user" {
		return perm != "settings" && perm != "users"
	}
	if role == "subuser" {
		return ParsePermissions(permissionsJSON).Has(perm)
	}
	return false
}
