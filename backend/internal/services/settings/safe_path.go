package settings

import (
	"crypto/rand"
	"math/big"
)

// GenerateSafePath returns a random 8-char entrance (letters + digits), unique per install.
func GenerateSafePath() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	const length = 8
	out := make([]byte, length)
	max := big.NewInt(int64(len(chars)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			out[i] = chars[i%len(chars)]
			continue
		}
		out[i] = chars[n.Int64()]
	}
	return string(out)
}

func safePathValue() string {
	return GenerateSafePath()
}
