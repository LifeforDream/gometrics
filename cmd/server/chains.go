package main

import (
	"crypto/rsa"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/LifeforDream/gometrics/internal/middlewares/logs"
	"github.com/LifeforDream/gometrics/internal/middlewares/mwcompress"
	"github.com/LifeforDream/gometrics/internal/middlewares/mwcrypto"
	"github.com/LifeforDream/gometrics/internal/middlewares/mwhash"
	"github.com/LifeforDream/gometrics/internal/middlewares/mwip"
)

type mwList = []func(http.Handler) http.Handler

type middlewares struct {
	stripSlashes  func(http.Handler) http.Handler
	logging       func(http.Handler) http.Handler
	clientIP      func(http.Handler) http.Handler
	trustedSubnet func(http.Handler) http.Handler
	hash          func(http.Handler) http.Handler
	crypto        func(http.Handler) http.Handler
	compress      func(http.Handler) http.Handler
}

// newMiddlewares создаёт настоящие мидлвары сервера по его настройкам.
func newMiddlewares(hashKey string, privateKey *rsa.PrivateKey, trusted *net.IPNet, logger *zap.Logger) middlewares {
	return middlewares{
		stripSlashes:  middleware.StripSlashes,
		logging:       logs.WithLogging(logger),
		clientIP:      mwip.WithClientIP,
		trustedSubnet: mwip.WithTrustedSubnet(trusted, logger),
		hash:          mwhash.WithHash(hashKey, logger),
		crypto:        mwcrypto.WithCrypto(privateKey, logger),
		compress:      mwcompress.Compress(logger),
	}
}

// buildChains раскладывает мидлвары по наборам для router.
// Это единственное место, где задаётся их состав и порядок.
// stripSlashes стоит в core, чтобы нормализовать URL до маршрутизации.
// trustedSubnet идёт первым в write, чтобы недоверенный клиент получал 403
// до дорогой проверки HMAC и RSA-расшифровки.
func buildChains(m middlewares) (core, read, write mwList) {
	shared := mwList{m.hash, m.crypto, m.compress}

	core = mwList{m.stripSlashes, m.logging, m.clientIP}
	read = shared
	write = append(mwList{m.trustedSubnet}, shared...)

	return core, read, write
}
