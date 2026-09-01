package audit

import (
	"encoding/json"
	"os"

	"go.uber.org/zap"
)

// FileAuditSender — реализация Sender, дописывающая каждое событие аудита
// JSON-строкой в конец файла.
type FileAuditSender struct {
	file   *os.File
	logger *zap.Logger
	c      chan Event
}

// NewFileAuditSender открывает (или создаёт) файл fn на дозапись и
// возвращает готовый к работе FileAuditSender.
func NewFileAuditSender(fn string, logger *zap.Logger) (*FileAuditSender, error) {
	file, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileAuditSender{file: file, logger: logger}, nil
}

func (as *FileAuditSender) setChan(c chan Event) {
	as.c = c
}

func (as *FileAuditSender) getID() string {
	return "file-audit-sender"
}

func (as *FileAuditSender) worker() {
	defer as.file.Close()
	enc := json.NewEncoder(as.file)

	for event := range as.c {
		if err := enc.Encode(event); err != nil {
			as.logger.Error("error encoding audit event", zap.Error(err))
		}
	}
}
