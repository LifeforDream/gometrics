// Package main является точкой запуска утилиты генерации простенького сертификата
package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"

	"github.com/LifeforDream/gometrics/pkg/certgen"
)

const (
	certFileMode = 0o644
	keyFileMode  = 0o600
)

func main() {
	certgenOptions, err := parseOptions()
	if err != nil {
		log.Fatal(err)
	}

	certIDBig, err := rand.Int(rand.Reader, big.NewInt(4096))
	if err != nil {
		log.Fatal(err)
	}
	certID := certIDBig.Int64()

	certBytes, keyBytes, err := certgen.GenerateKeyPair(certID, certgenOptions.Bits)

	if err != nil {
		log.Fatal(err)
	}

	certpath := filepath.Join(certgenOptions.OutDir, "cert.pem")

	if err = os.WriteFile(certpath, certBytes, certFileMode); err != nil {
		log.Fatal(err)
	}

	keypath := filepath.Join(certgenOptions.OutDir, "private.pem")

	if err = os.WriteFile(keypath, keyBytes, keyFileMode); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Wrote files:\n Certificate: %s \n Private key: %s", certpath, keypath)

}
