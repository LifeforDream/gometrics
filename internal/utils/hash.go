// Package utils содержит общие утилиты, не привязанные к конкретному
// слою: хэширование запросов, работу с контекстом, ретраи PostgreSQL
// и вспомогательные конструкторы указателей.
package utils

import "crypto/sha256"

// HashHeaderName — имя HTTP-заголовка, в котором передаётся
// hex-encoded HMAC-SHA256 тела запроса/ответа.
const HashHeaderName = "HashSHA256"

// GenSHA256 возвращает HMAC-SHA256 от d с ключом key (склеенные d и key
// хэшируются как один поток, без стандартного HMAC-конструирования).
func GenSHA256(d []byte, key string) []byte {
	h := sha256.New()
	h.Write(d)
	h.Write([]byte(key))
	return h.Sum(nil)
}
