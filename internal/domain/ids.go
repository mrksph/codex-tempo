package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/google/uuid"
)

func DeterministicID(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		h.Write([]byte(strings.TrimSpace(part)))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func DeterministicUUID(parts ...string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(strings.Join(parts, "\x00"))).String()
}
