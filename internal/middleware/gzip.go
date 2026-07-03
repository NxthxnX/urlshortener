package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	writer *gzip.Writer
}

func newGzipResponseWriter(w http.ResponseWriter) *gzipResponseWriter {
	return &gzipResponseWriter{
		ResponseWriter: w,
		writer:         gzip.NewWriter(w),
	}
}

func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	return w.writer.Write(p)
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		w.Header().Del("Content-Length")
		w.Header().Set("Content-Encoding", "gzip")
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Close() error {
	return w.writer.Close()
}

type gzipRequestReader struct {
	body   io.ReadCloser
	reader *gzip.Reader
}

func newGzipRequestReader(body io.ReadCloser) (*gzipRequestReader, error) {
	reader, err := gzip.NewReader(body)
	if err != nil {
		return nil, err
	}

	return &gzipRequestReader{
		body:   body,
		reader: reader,
	}, nil
}

func (r *gzipRequestReader) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *gzipRequestReader) Close() error {
	if err := r.body.Close(); err != nil {
		return err
	}
	return r.reader.Close()
}
