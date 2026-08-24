// Package compress содержит низкоуровневые обёртки поверх compress/gzip,
// используемые агентом при отправке батчей метрик и мидлваром mwcompress
// при (де)компрессии HTTP-тел.
package compress

import (
	"compress/gzip"
	"io"
)

// Writer — обёртка над gzip.Writer с максимальным уровнем сжатия
// (gzip.BestCompression). Close должен вызываться вызывающим кодом, чтобы
// сброшенные байты попали в нижележащий io.Writer.
type Writer struct {
	zw *gzip.Writer
}

// NewWriter создаёт Writer, пишущий сжатые данные в w.
func NewWriter(w io.Writer) (*Writer, error) {
	gw, err := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	return &Writer{zw: gw}, nil
}

func (w *Writer) Write(p []byte) (int, error) {
	return w.zw.Write(p)
}

// Close сбрасывает буфер gzip.Writer и должен вызываться после последней
// записи, иначе часть сжатых данных не попадёт в нижележащий io.Writer.
func (w *Writer) Close() error {
	return w.zw.Close()
}

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// NewCompressReader оборачивает r в gzip.Reader, прозрачно распаковывая
// читаемые данные. Close закрывает оба уровня: исходный r и gzip-reader.
func NewCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}
