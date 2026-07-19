package writer

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
)

// HDF5 filter label for DEFLATE / GZIP compression. Extracted as a
// constant so goconst doesn't flag the duplicate string across source +
// helper tests.
const filterDeflateName = "deflate"

// GZIPFilter implements GZIP compression (FilterID = 1).
// This filter uses the DEFLATE compression algorithm to reduce data size.
// In HDF5, this filter is named "deflate" following zlib terminology.
//
// Compression levels:
//
//	1 = fastest compression, larger files
//	6 = balanced (default)
//	9 = best compression, slower
type GZIPFilter struct {
	level int // Compression level (1-9)
}

// NewGZIPFilter creates a GZIP filter with the specified compression level.
//
// Valid levels:
//
//	1 = Fast compression, lower ratio
//	6 = Default (balanced)
//	9 = Best compression, slower
//
// Invalid levels are automatically adjusted to 6 (default).
func NewGZIPFilter(level int) *GZIPFilter {
	if level < 1 || level > 9 {
		level = 6 // Default compression level
	}
	return &GZIPFilter{level: level}
}

// ID returns the HDF5 filter identifier for GZIP.
func (f *GZIPFilter) ID() FilterID {
	return FilterGZIP
}

// Name returns the HDF5 filter name.
// HDF5 uses "deflate" (the underlying algorithm) rather than "gzip".
func (f *GZIPFilter) Name() string {
	return filterDeflateName
}

// Apply compresses data using zlib/DEFLATE algorithm (HDF5 GZIP format).
// Returns zlib-compressed data with zlib headers.
func (f *GZIPFilter) Apply(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	w, err := zlib.NewWriterLevel(&buf, f.level)
	if err != nil {
		return nil, fmt.Errorf("zlib writer creation failed: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("zlib compression failed: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("zlib close failed: %w", err)
	}

	return buf.Bytes(), nil
}

// Remove decompresses zlib-compressed data (HDF5 GZIP format).
// Returns the original uncompressed data.
func (f *GZIPFilter) Remove(data []byte) ([]byte, error) {
	buf := bytes.NewReader(data)

	r, err := zlib.NewReader(buf)
	if err != nil {
		return nil, fmt.Errorf("zlib reader creation failed: %w", err)
	}
	defer func() { _ = r.Close() }()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("zlib decompression failed: %w", err)
	}

	return decompressed, nil
}

// Encode returns the filter parameters for the Pipeline message.
//
// For GZIP, the client data contains a single value: the compression level.
// Flags are always 0 for GZIP.
func (f *GZIPFilter) Encode() (flags uint16, cdValues []uint32) {
	return 0, []uint32{uint32(f.level)} //nolint:gosec // G115: Compression level is 1-9, always fits in uint32
}
