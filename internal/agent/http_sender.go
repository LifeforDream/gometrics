package agent

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/LifeforDream/gometrics/internal/compress"
	"github.com/LifeforDream/gometrics/internal/crypto"
	models "github.com/LifeforDream/gometrics/internal/model"
	"github.com/LifeforDream/gometrics/internal/utils"
)

type httpBatchSender struct {
	serverAddress string
	hashKey       string
	publicKey     *rsa.PublicKey
	client        httpSender
	hostIP        string
}

func (s *httpBatchSender) Send(ctx context.Context, metrics []models.Metrics) error {
	var buf bytes.Buffer
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("error marshalling data: %w", err)
	}

	zw, err := compress.NewWriter(&buf)
	if err != nil {
		return fmt.Errorf("error creating compressor: %w", err)
	}

	_, err = zw.Write(jsonData)
	if err != nil {
		return fmt.Errorf("error writing compressed data: %w", err)
	}

	err = zw.Close()
	if err != nil {
		return fmt.Errorf("error closing compress writer: %w", err)
	}

	reqData := buf.Bytes()
	if s.publicKey != nil {
		reqData, err = crypto.Encrypt(s.publicKey, reqData)
		if err != nil {
			return fmt.Errorf("error encrypting data: %w", err)
		}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.serverAddress+"/updates", bytes.NewBuffer(reqData))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Encoding", "gzip")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Real-IP", s.hostIP)

	if s.hashKey != "" {
		hash := utils.GenSHA256(reqData, s.hashKey)
		request.Header.Set(utils.HashHeaderName, hex.EncodeToString(hash))
	}

	resp, err := s.client.Do(request)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}

// Close ничего не делает, кроме совместимости с интерфейсом batchSender.
func (s *httpBatchSender) Close() error {
	return nil
}
