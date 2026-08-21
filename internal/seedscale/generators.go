package seedscale

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/google/uuid"
)

func RandomHex(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return uuid.NewString()[:length]
	}
	return hex.EncodeToString(bytes)
}

func Sha256Hash(input string) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}

func RandomInt(min, max int) int {
	if max <= min {
		return min
	}
	diff := max - min
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(diff+1)))
	if err != nil {
		return min
	}
	return min + int(nBig.Int64())
}

func RandomDurationDaysAgo(days int) time.Time {
	offset := RandomInt(0, days*24*60)
	return time.Now().UTC().Add(-time.Duration(offset) * time.Minute)
}

func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func StringPtr(s string) *string {
	return &s
}

func TimePtr(t time.Time) *time.Time {
	return &t
}
