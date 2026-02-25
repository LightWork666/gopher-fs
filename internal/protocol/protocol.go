package protocol

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	DefaultTCPPort = ":9000"
	DiscoveryPort  = 9999
	BufferSize     = 1024
	DiscoveryMsg   = "DISCOVER_GOPHER_FS"
	
	// Operation Codes
	OpDownload = 1
	OpUpload   = 2
)

// FileHeader represents the metadata sent before file content
type FileHeader struct {
	FileNameLen uint32
	FileSize    int64
	Checksum    [32]byte
}

// ComputeChecksum calculates SHA256 hash of a file
func ComputeChecksum(filePath string) ([32]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return [32]byte{}, err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return [32]byte{}, err
	}

	var checksum [32]byte
	copy(checksum[:], hash.Sum(nil))
	return checksum, nil
}

// SendFileHeader sends the metadata over the connection
func SendFileHeader(w io.Writer, filename string, fileSize int64, checksum [32]byte) error {
	// 1. Send Filename Length
	if err := binary.Write(w, binary.LittleEndian, uint32(len(filename))); err != nil {
		return fmt.Errorf("failed to write filename length: %v", err)
	}
	
	// 2. Send File Size
	if err := binary.Write(w, binary.LittleEndian, fileSize); err != nil {
		return fmt.Errorf("failed to write file size: %v", err)
	}

	// 3. Send Checksum
	if _, err := w.Write(checksum[:]); err != nil {
		return fmt.Errorf("failed to write checksum: %v", err)
	}

	// 4. Send Filename
	if _, err := w.Write([]byte(filename)); err != nil {
		return fmt.Errorf("failed to write filename: %v", err)
	}

	return nil
}

// ReadFileHeader reads the metadata from the connection
func ReadFileHeader(r io.Reader) (string, int64, [32]byte, error) {
	// 1. Read Filename Length
	var nameLen uint32
	if err := binary.Read(r, binary.LittleEndian, &nameLen); err != nil {
		return "", 0, [32]byte{}, fmt.Errorf("failed to read filename length: %v", err)
	}

	// 2. Read File Size
	var fileSize int64
	if err := binary.Read(r, binary.LittleEndian, &fileSize); err != nil {
		return "", 0, [32]byte{}, fmt.Errorf("failed to read file size: %v", err)
	}

	// 3. Read Checksum
	var checksum [32]byte
	if _, err := io.ReadFull(r, checksum[:]); err != nil {
		return "", 0, [32]byte{}, fmt.Errorf("failed to read checksum: %v", err)
	}

	// 4. Read Filename
	nameBuf := make([]byte, nameLen)
	if _, err := io.ReadFull(r, nameBuf); err != nil {
		return "", 0, [32]byte{}, fmt.Errorf("failed to read filename: %v", err)
	}

	return string(nameBuf), fileSize, checksum, nil
}
