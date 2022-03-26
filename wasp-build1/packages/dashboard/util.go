package dashboard

import (
	"fmt"
	"time"

	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"github.com/iotaledger/wasp/packages/hashing"
	"github.com/iotaledger/wasp/packages/iscp/colored"
	"github.com/iotaledger/wasp/packages/kv"
)

func args(args ...interface{}) []interface{} {
	return args
}

func hashref(hash hashing.HashValue) *hashing.HashValue {
	return &hash
}

func colorref(color colored.Color) *colored.Color {
	return &color
}

func trim(max int, s string) string {
	if len(s) > max {
		s = s[:max] + "…"
	}
	// escape unprintable chars
	s = fmt.Sprintf("%q", s)
	// remove quotes
	return s[1 : len(s)-1]
}

func incUint32(n uint32) uint32 {
	return n + 1
}

func decUint32(n uint32) uint32 {
	return n - 1
}

func keyToString(k kv.Key) string {
	return string(k)
}

func bytesToString(b []byte) string {
	return string(b)
}

func anythingToString(i interface{}) string {
	if i == nil {
		return ""
	}
	return fmt.Sprintf("%v", i)
}

func formatTimestamp(ts interface{}) string {
	t, ok := ts.(time.Time)
	if !ok {
		t = time.Unix(0, ts.(int64))
	}
	return t.UTC().Format(time.RFC3339)
}

func formatTimestampOrNever(t time.Time) string {
	timestampNever := time.Time{}
	if t == timestampNever {
		return "NEVER"
	}
	return formatTimestamp(t)
}

func exploreAddressURL(baseURL string) func(address ledgerstate.Address) string {
	return func(address ledgerstate.Address) string {
		return baseURL + "/" + address.Base58()
	}
}
