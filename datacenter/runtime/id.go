package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

var fallbackIDCounter uint64

// GenerateID creates a request-safe identifier without adding a UUID
// dependency.
func GenerateID(prefix string) string {
	var random [16]byte
	if _, err := rand.Read(random[:]); err == nil {
		return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(random[:]))
	}
	sequence := atomic.AddUint64(&fallbackIDCounter, 1)
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), sequence)
}

func GenerateRequestID() string {
	return GenerateID("req")
}
