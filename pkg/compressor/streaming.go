package compressor

import (
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
)

// StreamingCompressor provides streaming compression without loading entire data into memory
type StreamingCompressor struct {
	writer *zstd.Encoder
	reader io.Reader
}

// NewStreamingCompressor creates a new streaming compressor
func NewStreamingCompressor(r io.Reader, level CompressionLevel) (*StreamingCompressor, error) {
	// Map compression level to zstd level
	var zstdLevel zstd.EncoderLevel
	switch level {
	case CompressionFastest:
		zstdLevel = zstd.SpeedFastest
	case CompressionDefault:
		zstdLevel = zstd.SpeedDefault
	case CompressionBest:
		zstdLevel = zstd.SpeedBestCompression
	default:
		zstdLevel = zstd.SpeedDefault
	}

	// Create pipe for streaming
	pr, pw := io.Pipe()

	// Create encoder
	encoder, err := zstd.NewWriter(pw, zstd.WithEncoderLevel(zstdLevel))
	if err != nil {
		return nil, fmt.Errorf("failed to create streaming encoder: %w", err)
	}

	sc := &StreamingCompressor{
		writer: encoder,
		reader: pr,
	}

	// Start compression in background
	go func() {
		defer pw.Close()
		defer encoder.Close()

		// Copy and compress data
		_, err := io.Copy(encoder, r)
		if err != nil {
			pw.CloseWithError(fmt.Errorf("compression error: %w", err))
			return
		}
	}()

	return sc, nil
}

// Read implements io.Reader interface
func (sc *StreamingCompressor) Read(p []byte) (int, error) {
	return sc.reader.Read(p)
}

// Close closes the streaming compressor
func (sc *StreamingCompressor) Close() error {
	if closer, ok := sc.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// StreamingDecompressor provides streaming decompression without loading entire data into memory
type StreamingDecompressor struct {
	reader *zstd.Decoder
}

// NewStreamingDecompressor creates a new streaming decompressor
func NewStreamingDecompressor(r io.Reader) (*StreamingDecompressor, error) {
	// Create decoder
	decoder, err := zstd.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create streaming decoder: %w", err)
	}

	return &StreamingDecompressor{
		reader: decoder,
	}, nil
}

// Read implements io.Reader interface
func (sd *StreamingDecompressor) Read(p []byte) (int, error) {
	return sd.reader.Read(p)
}

// Close closes the streaming decompressor
func (sd *StreamingDecompressor) Close() error {
	sd.reader.Close()
	return nil
}

// CompressStream creates a streaming compressor that compresses data on-the-fly
func CompressStream(r io.Reader, level CompressionLevel) (io.ReadCloser, error) {
	if level == CompressionNone {
		return io.NopCloser(r), nil
	}

	return NewStreamingCompressor(r, level)
}

// DecompressStream creates a streaming decompressor that decompresses data on-the-fly
func DecompressStream(r io.Reader) (io.ReadCloser, error) {
	return NewStreamingDecompressor(r)
}
