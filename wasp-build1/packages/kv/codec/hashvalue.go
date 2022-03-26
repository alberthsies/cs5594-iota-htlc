package codec

import (
	"github.com/iotaledger/wasp/packages/hashing"
	"golang.org/x/xerrors"
)

func DecodeHashValue(b []byte, def ...hashing.HashValue) (hashing.HashValue, error) {
	if b == nil {
		if len(def) == 0 {
			return hashing.HashValue{}, xerrors.Errorf("cannot decode nil bytes")
		}
		return def[0], nil
	}
	return hashing.HashValueFromBytes(b)
}

func EncodeHashValue(value hashing.HashValue) []byte {
	return value[:]
}
