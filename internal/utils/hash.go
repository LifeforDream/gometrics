package utils

import "crypto/sha256"

const HashHeaderName = "HashSHA256"

func GenSHA256(d []byte, key string) []byte {
	h := sha256.New()
	h.Write(d)
	h.Write([]byte(key))
	return h.Sum(nil)
}
