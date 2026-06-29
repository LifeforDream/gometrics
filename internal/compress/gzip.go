package compress

import (
	"compress/gzip"
	"io"
)

type Writer struct {
	zw *gzip.Writer
}

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

func (w *Writer) Close() error {
	return w.zw.Close()
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
