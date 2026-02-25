package main

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"

	"flag"
	"fmt"
	"io"
	"log"
	
	"crypto/tls"
	"encoding/binary"
	"os"
	"path/filepath"
	"time"

	"gopher-fs/internal/discovery"
	"gopher-fs/internal/protocol"
	"gopher-fs/internal/security"
	"gopher-fs/internal/ui"
)

func main() {
	filename := flag.String("file", "", "File name to request or upload")
	upload := flag.Bool("upload", false, "Upload file instead of downloading")
	flag.Parse()

	if *filename == "" {
		fmt.Println("Usage: client -file [filename] [-upload]")
		return
	}

	startClient(*filename, *upload)
}

func startClient(filename string, upload bool) {
	serverAddr := discovery.FindServer()
	if serverAddr == "" {
		log.Fatal("No servers found. Discovery failed or timed out.")
	}
	
	if upload {
		uploadFile(serverAddr, filename)
	} else {
		downloadFile(serverAddr, filename)
	}
}

func uploadFile(serverAddr, filename string) {
	// 1. Establish Secure Connection
	tlsConfig, err := security.GenerateTLSConfig()
	if err != nil {
		log.Fatalf("Error improved security configuration: %v", err)
	}

	conn, err := tls.Dial("tcp", serverAddr, tlsConfig)
	if err != nil {
		log.Fatalf("Error connecting to server (TLS): %v", err)
	}
	defer conn.Close()
	
	log.Printf("Connected to server for upload: %s", serverAddr)

	// 2. Send Operation Code (Upload)
	opCode := uint8(protocol.OpUpload)
	if err := binary.Write(conn, binary.LittleEndian, opCode); err != nil {
		log.Fatalf("Error sending operation code: %v", err)
	}

	// 3. Open Local File
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Error opening file %s: %v", filename, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatalf("Error getting file info: %v", err)
	}

	// 4. Compute Checksum
	log.Println("Computing checksum...")
	checksum, err := protocol.ComputeChecksum(filename)
	if err != nil {
		log.Fatalf("Error computing checksum: %v", err)
	}

	// 5. Send Header
	log.Printf("Sending file header (Size: %d bytes)", fileInfo.Size())
	err = protocol.SendFileHeader(conn, filepath.Base(filename), fileInfo.Size(), checksum)
	if err != nil {
		log.Fatalf("Error sending file header: %v", err)
	}

	// 6. Stream File Content
	pw := ui.NewProgressWriter(fileInfo.Size(), conn)
	sentBytes, err := io.Copy(pw, file)
	if err != nil {
		log.Fatalf("Error sending file data: %v", err)
	}
	log.Printf("Successfully uploaded %s (%d bytes)", filename, sentBytes)
}

func downloadFile(serverAddr, filename string) {
	// 1. Establish Secure Connection
	tlsConfig, err := security.GenerateTLSConfig()
	if err != nil {
		log.Fatalf("Error improved security configuration: %v", err)
	}

	conn, err := tls.Dial("tcp", serverAddr, tlsConfig)
	if err != nil {
		log.Fatalf("Error connecting to server (TLS): %v", err)
	}
	defer conn.Close()

	// 2. Send Operation Code (Download)
	opCode := uint8(protocol.OpDownload)
	if err := binary.Write(conn, binary.LittleEndian, opCode); err != nil {
		log.Fatalf("Error sending operation code: %v", err)
	}

	// 3. Send Request (Filename)
	log.Printf("Requesting file: %s", filename)
	nameBytes := []byte(filename)
	nameLen := uint32(len(nameBytes))
	
	// Send length
	if err := binary.Write(conn, binary.LittleEndian, nameLen); err != nil {
		log.Fatalf("Error sending filename length: %v", err)
	}
	// Send name
	if _, err := conn.Write(nameBytes); err != nil {
		log.Fatalf("Error sending filename: %v", err)
	}

	// 2. Read Response Header (Metadata)
	log.Println("Waiting for response...")
	serverFileName, fileSize, serverChecksum, err := protocol.ReadFileHeader(conn)
	if err != nil {
		log.Fatalf("Error reading file header: %v", err)
	}

	fmt.Printf("File Found: %s (%d bytes)\n", serverFileName, fileSize)
	fmt.Printf("Server Checksum: %x\n", serverChecksum)

	// 3. Download File Content
	outputFile := "downloaded_" + filepath.Base(filename)
	outFile, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Error creating local file: %v", err)
	}
	defer outFile.Close()

	// Create a TeeReader to compute checksum while downloading
	hasher := sha256.New()
	
	// Chain: Network -> ProgressReader -> LimitReader -> TeeReader
	// We want progress to update as bytes come off the wire.
	
	progReader := ui.NewProgressReader(fileSize, conn)
	limitReader := io.LimitReader(progReader, fileSize)
	tee := io.TeeReader(limitReader, hasher)

	startTime := time.Now()
	// Copy to File from the TeeReader (which splits to Hasher)
	receivedBytes, err := io.Copy(outFile, tee)
	if err != nil {
		log.Fatalf("Error downloading file: %v", err)
	}
	duration := time.Since(startTime)

	// 4. Verify Checksum
	var clientChecksum [32]byte
	copy(clientChecksum[:], hasher.Sum(nil))
	
	fmt.Println() // Clear progress bar line
	fmt.Printf("Downloaded %d bytes in %v\n", receivedBytes, time.Since(startTime))
	fmt.Printf("Client Checksum: %x\n", clientChecksum)

	if clientChecksum == serverChecksum {
		fmt.Println("✅ Integrity Verified: Checksum matches!")
	} else {
		fmt.Println("❌ Integrity Failure: Checksum mismatch!")
		os.Remove(outputFile) // Delete corrupted file? Or define policy.
	}
}
