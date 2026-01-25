package compressor

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestCompressDecompress(t *testing.T) {
	tests := []struct {
		name  string
		data  string
		level CompressionLevel
	}{
		{
			name:  "small text with default compression",
			data:  "Hello, World!",
			level: CompressionDefault,
		},
		{
			name:  "large text with fastest compression",
			data:  strings.Repeat("This is a test string for compression. ", 100),
			level: CompressionFastest,
		},
		{
			name:  "json data with best compression",
			data:  `{"name":"John","age":30,"city":"New York","items":["item1","item2","item3"]}`,
			level: CompressionBest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compress
			compressed, err := CompressBytes([]byte(tt.data), tt.level)
			if err != nil {
				t.Fatalf("CompressBytes failed: %v", err)
			}

			// Verify compressed data is smaller (for large enough data)
			if len(tt.data) > 100 && len(compressed) >= len(tt.data) {
				t.Logf("Warning: compressed size (%d) >= original size (%d)", len(compressed), len(tt.data))
			}

			// Decompress
			decompressed, err := DecompressBytes(compressed)
			if err != nil {
				t.Fatalf("DecompressBytes failed: %v", err)
			}

			// Verify data matches
			if string(decompressed) != tt.data {
				t.Errorf("Decompressed data doesn't match original.\nExpected: %s\nGot: %s", tt.data, string(decompressed))
			}
		})
	}
}

func TestCompressDecompressReader(t *testing.T) {
	data := "This is test data for reader compression"

	// Compress
	reader := strings.NewReader(data)
	compressedReader, err := Compress(reader, CompressionDefault)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}
	defer compressedReader.Close()

	// Read compressed data
	compressed, err := io.ReadAll(compressedReader)
	if err != nil {
		t.Fatalf("Failed to read compressed data: %v", err)
	}

	// Decompress
	decompressedReader, err := Decompress(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	defer decompressedReader.Close()

	// Read decompressed data
	decompressed, err := io.ReadAll(decompressedReader)
	if err != nil {
		t.Fatalf("Failed to read decompressed data: %v", err)
	}

	// Verify
	if string(decompressed) != data {
		t.Errorf("Decompressed data doesn't match.\nExpected: %s\nGot: %s", data, string(decompressed))
	}
}

func TestIsCompressed(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		compressed bool
	}{
		{
			name:       "uncompressed text",
			data:       []byte("Hello, World!"),
			compressed: false,
		},
		{
			name:       "empty data",
			data:       []byte{},
			compressed: false,
		},
		{
			name:       "short data",
			data:       []byte{0x28, 0xB5},
			compressed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCompressed(tt.data)
			if result != tt.compressed {
				t.Errorf("IsCompressed() = %v, want %v", result, tt.compressed)
			}
		})
	}

	// Test with actually compressed data
	t.Run("compressed data", func(t *testing.T) {
		original := []byte("This is test data that will be compressed")
		compressed, err := CompressBytes(original, CompressionDefault)
		if err != nil {
			t.Fatalf("CompressBytes failed: %v", err)
		}

		if !IsCompressed(compressed) {
			t.Error("IsCompressed() should return true for compressed data")
		}
	})
}

func TestCompressionLevels(t *testing.T) {
	data := []byte(strings.Repeat("This is a test string for compression level comparison. ", 100))

	levels := []CompressionLevel{
		CompressionFastest,
		CompressionDefault,
		CompressionBest,
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			compressed, err := CompressBytes(data, level)
			if err != nil {
				t.Fatalf("CompressBytes failed: %v", err)
			}

			t.Logf("Level %s: Original size: %d, Compressed size: %d, Ratio: %.2f%%",
				level.String(), len(data), len(compressed),
				float64(len(compressed))/float64(len(data))*100)

			// Verify decompression works
			decompressed, err := DecompressBytes(compressed)
			if err != nil {
				t.Fatalf("DecompressBytes failed: %v", err)
			}

			if !bytes.Equal(decompressed, data) {
				t.Error("Decompressed data doesn't match original")
			}
		})
	}
}

func TestCompressionNone(t *testing.T) {
	data := []byte("Test data")

	// Compress with CompressionNone
	compressed, err := CompressBytes(data, CompressionNone)
	if err != nil {
		t.Fatalf("CompressBytes failed: %v", err)
	}

	// Should return original data
	if !bytes.Equal(compressed, data) {
		t.Error("CompressionNone should return original data")
	}
}

func (cl CompressionLevel) String() string {
	switch cl {
	case CompressionNone:
		return "None"
	case CompressionFastest:
		return "Fastest"
	case CompressionDefault:
		return "Default"
	case CompressionBest:
		return "Best"
	default:
		return "Unknown"
	}
}

func BenchmarkCompress(b *testing.B) {
	data := []byte(strings.Repeat("This is benchmark data for compression testing. ", 1000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CompressBytes(data, CompressionDefault)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecompress(b *testing.B) {
	data := []byte(strings.Repeat("This is benchmark data for decompression testing. ", 1000))
	compressed, err := CompressBytes(data, CompressionDefault)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DecompressBytes(compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}
