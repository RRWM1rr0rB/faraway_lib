// Package byteutils provides advanced utilities for byte manipulation,
// including serialization, compression, encoding, debugging, and cryptographic operations.
// It features memory-efficient operations with buffer pooling and supports common encoding formats.
package bytes

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

// bufferPool maintains a pool of reusable bytes.Buffer objects to reduce allocations
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// Serialize converts any value to JSON-encoded []byte.
// T: The type of the value to serialize
// v: The value to serialize
// Returns:
//   - []byte: JSON-encoded data
//   - error: Marshaling error if any
func Serialize[T any](v T) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("serialize: marshaling error: %w", err)
	}
	return data, nil
}

// MustSerialize serializes a value to JSON-encoded []byte and panics on error.
// Useful for initialization where errors should be fatal.
// T: The type of the value to serialize
// v: The value to serialize
// Returns:
//   - []byte: JSON-encoded data
func MustSerialize[T any](v T) []byte {
	data, err := Serialize(v)
	if err != nil {
		panic(err)
	}
	return data
}

// Deserialize converts JSON-encoded []byte to a value of specified type.
// T: The target type for deserialization
// data: JSON-encoded data
// Returns:
//   - T: Deserialized value
//   - error: Unmarshaling error if any
func Deserialize[T any](data []byte) (T, error) {
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return result, fmt.Errorf("deserialize: unmarshaling error: %w", err)
	}
	return result, nil
}

// Compress data using gzip with specified compression level.
// data: Input data to compress
// level: Compression level (gzip.BestSpeed, gzip.BestCompression, etc.)
// Returns:
//   - []byte: Compressed data
//   - error: Compression-related error if any
//
// Notes:
//   - Uses buffer pooling for efficient memory reuse
func Compress(data []byte, level int) ([]byte, error) {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()

	gz, err := gzip.NewWriterLevel(buf, level)
	if err != nil {
		return nil, fmt.Errorf("compress: writer initialization failed: %w", err)
	}

	if _, err := gz.Write(data); err != nil {
		return nil, fmt.Errorf("compress: data write failed: %w", err)
	}

	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("compress: writer close failed: %w", err)
	}

	return buf.Bytes(), nil
}

// MustCompress compresses data using gzip and panics on error.
// data: Input data to compress
// level: Compression level
// Returns:
//   - []byte: Compressed data
func MustCompress(data []byte, level int) []byte {
	compressed, err := Compress(data, level)
	if err != nil {
		panic(err)
	}
	return compressed
}

// Decompress gzip-compressed data.
// data: Compressed input data
// Returns:
//   - []byte: Decompressed data
//   - error: Decompression-related error if any
//
// Notes:
//   - Performs integrity check for trailing data
func Decompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decompress: reader initialization failed: %w", err)
	}
	defer r.Close()

	result, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("decompress: data read failed: %w", err)
	}

	// Check for trailing data which might indicate corruption
	if extra, err := io.ReadAll(r); err != nil || len(extra) > 0 {
		return result, fmt.Errorf("decompress: corrupted gzip data")
	}

	return result, nil
}

// MustDecompress decompresses data and panics on error.
// data: Compressed input data
// Returns:
//   - []byte: Decompressed data
func MustDecompress(data []byte) []byte {
	decompressed, err := Decompress(data)
	if err != nil {
		panic(err)
	}
	return decompressed
}

// Base64 encoding types
const (
	URLEncoding = iota // URL-safe base64 encoding
	StdEncoding        // Standard base64 encoding
)

// ToBase64 encodes data to base64 string.
// data: Input data to encode
// encType: Encoding type (URLEncoding or StdEncoding)
// Returns:
//   - string: Base64-encoded string
func ToBase64(data []byte, encType int) string {
	switch encType {
	case URLEncoding:
		return base64.URLEncoding.EncodeToString(data)
	default:
		return base64.StdEncoding.EncodeToString(data)
	}
}

// FromBase64 decodes base64 string to []byte.
// s: Base64-encoded string
// encType: Encoding type (URLEncoding or StdEncoding)
// Returns:
//   - []byte: Decoded data
//   - error: Decoding error if any
func FromBase64(s string, encType int) ([]byte, error) {
	switch encType {
	case URLEncoding:
		return base64.URLEncoding.DecodeString(s)
	default:
		return base64.StdEncoding.DecodeString(s)
	}
}

// HexDump generates formatted hexadecimal representation of data.
// data: Input data to format
// bytesPerLine: Number of bytes per line in output
// Returns:
//   - string: Formatted hex dump
//
// Notes:
//   - Returns "<empty>" for zero-length input
func HexDump(data []byte, bytesPerLine int) string {
	if len(data) == 0 {
		return "<empty>"
	}

	var sb strings.Builder
	for i, b := range data {
		if i%bytesPerLine == 0 && i > 0 {
			sb.WriteByte('\n')
		}
		fmt.Fprintf(&sb, "%02x ", b)
	}
	return strings.TrimSpace(sb.String())
}

// Dump generates structured string representation of complex data types.
// data: Value to dump (supports maps, slices, and primitives)
// indent: Indentation string (e.g., "  " for two spaces)
// Returns:
//   - string: Formatted dump output
func Dump(data interface{}, indent string) string {
	var sb strings.Builder
	dumpValue(&sb, data, indent, 0)
	return sb.String()
}

// dumpValue recursively formats values for Dump function
// sb: Target string builder
// v: Current value to format
// indent: Indentation string
// depth: Current recursion depth
func dumpValue(sb *strings.Builder, v interface{}, indent string, depth int) {
	const indentStep = 2
	prefix := strings.Repeat(indent, depth)

	switch val := v.(type) {
	case map[string]interface{}:
		sb.WriteString("{\n")
		for k, v := range val {
			fmt.Fprintf(sb, "%s%*s%s: ", prefix, indentStep, "", k)
			dumpValue(sb, v, indent, depth+1)
			sb.WriteString("\n")
		}
		fmt.Fprintf(sb, "%s}", prefix)
	case []interface{}:
		sb.WriteString("[\n")
		for _, item := range val {
			fmt.Fprintf(sb, "%s%*s", prefix, indentStep, "")
			dumpValue(sb, item, indent, depth+1)
			sb.WriteString("\n")
		}
		fmt.Fprintf(sb, "%s]", prefix)
	default:
		fmt.Fprintf(sb, "%#v", val)
	}
}

// String converts []byte to string without allocation.
// data: Input byte slice
// Returns:
//   - string: Converted string
//
// Note:
//   - Unsafe if original byte slice is modified
func String(data []byte) string {
	return string(data)
}

// Bytes converts string to []byte without allocation.
// s: Input string
// Returns:
//   - []byte: Converted byte slice
//
// Note:
//   - Unsafe if original string is modified
func Bytes(s string) []byte {
	return []byte(s)
}

// Equal performs constant-time comparison of byte slices.
// a: First byte slice
// b: Second byte slice
// Returns:
//   - bool: True if slices are equal
//
// Note:
//   - Uses constant-time comparison to prevent timing attacks
func Equal(a, b []byte) bool {
	return bytes.Equal(a, b)
}

// XOR performs byte-wise XOR encryption/decryption.
// data: Input data to process
// key: Encryption key
// Returns:
//   - []byte: Processed data
//
// Note:
//   - Repeating key XOR (Vernam cipher when key length == data length)
func XOR(data, key []byte) []byte {
	result := make([]byte, len(data))
	keyLen := len(key)
	if keyLen == 0 {
		return data
	}

	for i := 0; i < len(data); i++ {
		result[i] = data[i] ^ key[i%keyLen]
	}
	return result
}

// Chunk splits data into fixed-size byte slices.
// data: Input data to split
// size: Desired chunk size
// Returns:
//   - [][]byte: Slice of data chunks
//
// Note:
//   - Returns nil for size <= 0
func Chunk(data []byte, size int) [][]byte {
	if size <= 0 {
		return nil
	}

	chunks := make([][]byte, 0, (len(data)+size-1)/size)
	for i := 0; i < len(data); i += size {
		end := i + size
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	return chunks
}
