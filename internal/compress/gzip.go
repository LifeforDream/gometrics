package compress

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"slices"
)

var encodableTypes = []string{"application/json", "text/html"}

type compressWriter struct {
	w           http.ResponseWriter
	zw          *gzip.Writer
	acceptsGzip bool
	wroteGzip   bool
}

func NewCompressWriter(w http.ResponseWriter, acceptsGzip bool) *compressWriter {
	return &compressWriter{
		w:           w,
		zw:          gzip.NewWriter(w),
		acceptsGzip: acceptsGzip,
	}
}

func (c *compressWriter) toEncode(w http.ResponseWriter) bool {
	return slices.Contains(encodableTypes, w.Header().Get("Content-Type")) && c.acceptsGzip
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	if c.toEncode(c.w) {
		c.wroteGzip = true
		// in case of missing WriteHeader() in handler
		c.w.Header().Set("Content-Encoding", "gzip")
		return c.zw.Write(p)
	}
	return c.w.Write(p)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	// only works if w.Header().Set() was called before WriteHeader()
	if c.toEncode(c.w) && statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

func (c *compressWriter) Close() error {
	if c.wroteGzip {
		return c.zw.Close()
	}
	return nil
}

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

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

func Buffer(b *bytes.Buffer) error {
	od, err := io.ReadAll(b)
	b.Reset()

	if err != nil {
		return fmt.Errorf("failed to read from uncompressed data: %w", err)
	}
	w, err := gzip.NewWriterLevel(b, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("failed init compress writer: %w", err)
	}
	_, err = w.Write(od)
	if err != nil {
		return fmt.Errorf("failed write data to compress temporary buffer: %w", err)
	}
	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed compress data: %w", err)
	}
	return nil
}
