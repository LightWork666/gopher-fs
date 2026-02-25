package main

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"

	"gopher-fs/internal/discovery"
	"gopher-fs/internal/protocol"
	"gopher-fs/internal/security"
)

func main() {
	// Start Discovery Listener
	go discovery.Listen(protocol.DefaultTCPPort)

	// Configure TLS
	tlsConfig, err := security.GenerateTLSConfig()
	if err != nil {
		log.Fatalf("Error configuring TLS: %v", err)
	}

	// Start Secure TCP File Server
	listener, err := tls.Listen("tcp", protocol.DefaultTCPPort, tlsConfig)
	if err != nil {
		log.Fatalf("Error starting TCP server: %v", err)
	}
	defer listener.Close()

	fmt.Printf("Secure File Server listening on %s (TLS enabled)\n", protocol.DefaultTCPPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	log.Printf("Accepted connection from %s", conn.RemoteAddr())

	// 1. Read Operation Code (1 byte)
	var opCode uint8
	if err := binary.Read(conn, binary.LittleEndian, &opCode); err != nil {
		log.Printf("Error reading operation code: %v", err)
		return
	}

	switch opCode {
	case protocol.OpDownload:
		handleDownload(conn)
	case protocol.OpUpload:
		handleUpload(conn)
	default:
		log.Printf("Unknown operation code: %d", opCode)
	}
}

func handleDownload(conn net.Conn) {
	// 2. Read requested filename length
	var nameLen uint32
	if err := binary.Read(conn, binary.LittleEndian, &nameLen); err != nil {
		log.Printf("Error reading filename length: %v", err)
		return
	}

	// 3. Read filename
	nameBuf := make([]byte, nameLen)
	if _, err := io.ReadFull(conn, nameBuf); err != nil {
		log.Printf("Error reading filename: %v", err)
		return
	}
	fileName := string(nameBuf)
	
	// Sanitize filename
	cleanedFileName := filepath.Base(fileName)
	log.Printf("Client requested file: %s", cleanedFileName)

	// 4. Open File
	file, err := os.Open(cleanedFileName)
	if err != nil {
		log.Printf("Error opening file %s: %v", cleanedFileName, err)
		return
	}
	defer file.Close()

	// 5. Get File Info (Size)
	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("Error getting file info: %v", err)
		return
	}

	// 6. Compute Checksum
	log.Println("Computing checksum...")
	checksum, err := protocol.ComputeChecksum(cleanedFileName)
	if err != nil {
		log.Printf("Error computing checksum: %v", err)
		return
	}

	// 7. Send Header (File Metadata)
	log.Printf("Sending file header (Size: %d bytes)", fileInfo.Size())
	err = protocol.SendFileHeader(conn, cleanedFileName, fileInfo.Size(), checksum)
	if err != nil {
		log.Printf("Error sending file header: %v", err)
		return
	}

	// 8. Stream File Content
	sentBytes, err := io.Copy(conn, file)
	if err != nil {
		log.Printf("Error sending file data: %v", err)
		return
	}
	log.Printf("Sent %d bytes for file %s", sentBytes, cleanedFileName)
}

func handleUpload(conn net.Conn) {
	log.Println("Client initiating upload...")

	// 1. Read Header
	fileName, fileSize, checksum, err := protocol.ReadFileHeader(conn) // Corrected: Receive header first
	if err != nil {
		log.Printf("Error reading upload header: %v", err)
		return
	}
	log.Printf("Receiving file: %s (%d bytes)", fileName, fileSize)

	// 2. Create File
	savePath := "server_" + filepath.Base(fileName)
	file, err := os.Create(savePath)
	if err != nil {
		log.Printf("Error creating file %s: %v", savePath, err)
		return
	}
	defer file.Close()

	// 3. Stream Data
	// In a real upload, we read exactly 'fileSize' bytes.
	receivedBytes, err := io.CopyN(file, conn, fileSize)
	if err != nil {
		if err != io.EOF {
			log.Printf("Error receiving file data: %v", err)
			return
		}
	}

	// 4. Verify Checksum
	localChecksum, err := protocol.ComputeChecksum(savePath)
	if err != nil {
		log.Printf("Error computing local checksum: %v", err)
		return
	}

	if localChecksum == checksum {
		log.Printf("Successfully received %s (%d bytes). Integrity Verified.", savePath, receivedBytes)
	} else {
		log.Printf("WARNING: Checksum mismatch for %s", savePath)
	}
}
