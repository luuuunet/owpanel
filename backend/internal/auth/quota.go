package auth

// QuotaMBFromBytes converts byte size to billed megabytes (minimum 1 MB when bytes > 0).
func QuotaMBFromBytes(bytes int64) int64 {
	if bytes <= 0 {
		return 0
	}
	mb := bytes / (1024 * 1024)
	if mb == 0 {
		return 1
	}
	return mb
}
