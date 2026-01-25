package compressor

import (
	"bytes"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
)

// CompressionLevel defines the compression level for Zstd
type CompressionLevel int

const (
	// CompressionNone indicates no compression
	CompressionNone CompressionLevel = 0
	// CompressionFastest provides fastest compression with lower ratio
	CompressionFastest CompressionLevel = 1
	// CompressionDefault provides balanced compression
	CompressionDefault CompressionLevel = 2
	// CompressionBest provides best compression ratio but slower
	CompressionBest CompressionLevel = 3
)

// Compressor defines the interface for compression operations
type Compressor interface {
	// Compress compresses data from reader and returns compressed reader
	Compress(r io.Reader, level CompressionLevel) (io.ReadCloser, error)

	// Decompress decompresses data from reader and returns decompressed reader
	Decompress(r io.Reader) (io.ReadCloser, error)

	// CompressBytes compresses byte slice and returns compressed bytes
	CompressBytes(data []byte, level CompressionLevel) ([]byte, error)

	// DecompressBytes decompresses byte slice and returns decompressed bytes
	DecompressBytes(data []byte) ([]byte, error)

	// IsCompressed checks if data is compressed by checking magic bytes
	IsCompressed(data []byte) bool
}

// zstdCompressor implements Compressor interface using Zstd
type zstdCompressor struct {
	encoderPool *zstd.Encoder
	decoderPool *zstd.Decoder
}

// NewZstdCompressor creates a new Zstd compressor
func NewZstdCompressor() (Compressor, error) {
	// Create encoder with default settings
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
	}

	// Create decoder with default settings
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
	}

	return &zstdCompressor{
		encoderPool: encoder,
		decoderPool: decoder,
	}, nil
}

// Compress compresses data from reader using Zstd
func (c *zstdCompressor) Compress(r io.Reader, level CompressionLevel) (io.ReadCloser, error) {
	if level == CompressionNone {
		// No compression, return as-is wrapped in NopCloser
		return io.NopCloser(r), nil
	}

	// Read all data from reader
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	// Compress data
	compressed, err := c.CompressBytes(data, level)
	if err != nil {
		return nil, err
	}

	// Return compressed data as reader
	return io.NopCloser(bytes.NewReader(compressed)), nil
}

// Decompress decompresses data from reader using Zstd
func (c *zstdCompressor) Decompress(r io.Reader) (io.ReadCloser, error) {
	// Read all data from reader
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	// Check if data is actually compressed
	if !c.IsCompressed(data) {
		// Not compressed, return as-is
		return io.NopCloser(bytes.NewReader(data)), nil
	}

	// Decompress data
	decompressed, err := c.DecompressBytes(data)
	if err != nil {
		return nil, err
	}

	// Return decompressed data as reader
	return io.NopCloser(bytes.NewReader(decompressed)), nil
}

// CompressBytes compresses byte slice using Zstd
func (c *zstdCompressor) CompressBytes(data []byte, level CompressionLevel) ([]byte, error) {
	if level == CompressionNone {
		return data, nil
	}

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

	// Create encoder with specified level
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstdLevel))
	if err != nil {
		return nil, fmt.Errorf("failed to create encoder: %w", err)
	}
	defer encoder.Close()

	// Compress data
	compressed := encoder.EncodeAll(data, make([]byte, 0, len(data)))

	return compressed, nil
}

// DecompressBytes decompresses byte slice using Zstd
func (c *zstdCompressor) DecompressBytes(data []byte) ([]byte, error) {
	// Check if data is compressed
	if !c.IsCompressed(data) {
		return data, nil
	}

	// Create decoder
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}
	defer decoder.Close()

	// Decompress data
	decompressed, err := decoder.DecodeAll(data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress: %w", err)
	}

	return decompressed, nil
}

// IsCompressed checks if data is Zstd compressed by checking magic bytes
func (c *zstdCompressor) IsCompressed(data []byte) bool {
	// Zstd magic number: 0x28, 0xB5, 0x2F, 0xFD
	if len(data) < 4 {
		return false
	}

	return data[0] == 0x28 && data[1] == 0xB5 && data[2] == 0x2F && data[3] == 0xFD
}

// Default global compressor instance
var defaultCompressor Compressor

func init() {
	var err error
	defaultCompressor, err = NewZstdCompressor()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize default compressor: %v", err))
	}
}

// Compress compresses data using the default compressor
func Compress(r io.Reader, level CompressionLevel) (io.ReadCloser, error) {
	return defaultCompressor.Compress(r, level)
}

// Decompress decompresses data using the default compressor
func Decompress(r io.Reader) (io.ReadCloser, error) {
	return defaultCompressor.Decompress(r)
}

// CompressBytes compresses byte slice using the default compressor
func CompressBytes(data []byte, level CompressionLevel) ([]byte, error) {
	return defaultCompressor.CompressBytes(data, level)
}

// DecompressBytes decompresses byte slice using the default compressor
func DecompressBytes(data []byte) ([]byte, error) {
	return defaultCompressor.DecompressBytes(data)
}

// IsCompressed checks if data is compressed using the default compressor
func IsCompressed(data []byte) bool {
	return defaultCompressor.IsCompressed(data)
}
