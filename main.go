package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultTCPPort = ":9000"
	discoveryPort  = 9999
	bufferSize     = 1024
)

func main() {
	mode := flag.String("mode", "", "Mode to run: 'server' or 'client'")
	filename := flag.String("file", "", "File name to request (client mode)")
	flag.Parse()

	if *mode == "server" {
		startServer()
	} else if *mode == "client" {
		if *filename == "" {
			fmt.Println("Please provide a file name using -file")
			return
		}
		startClient(*filename)
	} else {
		fmt.Println("Usage: gopher-fs -mode [server|client] -file [filename]")
	}
}

func startServer() {
	// Start Discovery Listener
	go startDiscoveryServer()

	// Start TCP File Server
	listener, err := net.Listen("tcp", defaultTCPPort)
	if err != nil {
		log.Fatalf("Error starting TCP server: %v", err)
	}
	defer listener.Close()

	fmt.Printf("File Server listening on %s\n", defaultTCPPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

func startDiscoveryServer() {
	addr := &net.UDPAddr{
		Port: discoveryPort,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		log.Fatalf("Error starting UDP discovery server: %v", err)
	}
	defer conn.Close()

	fmt.Printf("Discovery Server listening on UDP %d\n", discoveryPort)

	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error reading UDP: %v", err)
			continue
		}
		
		msg := string(buf[:n])
		if msg == "DISCOVER_GOPHER_FS" {
			log.Printf("Received discovery request from %s", remoteAddr)
			// Respond with our TCP port
			_, err := conn.WriteToUDP([]byte(defaultTCPPort), remoteAddr)
			if err != nil {
				log.Printf("Error sending discovery response: %v", err)
			}
		}
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	log.Printf("Accepted connection from %s", conn.RemoteAddr())

	// Read file name length (4 bytes)
	var nameLen uint32
	if err := binary.Read(conn, binary.LittleEndian, &nameLen); err != nil {
		log.Printf("Error reading file name length: %v", err)
		return
	}

	// Read file name
	nameBuf := make([]byte, nameLen)
	if _, err := io.ReadFull(conn, nameBuf); err != nil {
		log.Printf("Error reading file name: %v", err)
		return
	}
	fileName := string(nameBuf)
	
	// Sanitize filename to prevent directory traversal
	cleanedFileName := filepath.Base(fileName)
	log.Printf("Client requested file: %s", cleanedFileName)

	file, err := os.Open(cleanedFileName)
	if err != nil {
		log.Printf("Error opening file %s: %v", cleanedFileName, err)
		// Optionally send error to client, but for simplicity we just close
		return
	}
	defer file.Close()

	// Stream file to client
	sentBytes, err := io.Copy(conn, file)
	if err != nil {
		log.Printf("Error sending file: %v", err)
		return
	}
	log.Printf("Sent %d bytes for file %s", sentBytes, cleanedFileName)
}

func startClient(filename string) {
	serverAddr := discoverServer()
	if serverAddr == "" {
		log.Println("No servers found. Discovery failed or timed out.")
		return
	}
	
	downloadFile(serverAddr, filename)
}

func discoverServer() string {
	fmt.Println("Broadcasting for servers...")

	// Listen on a random UDP port for the response (Force IPv4)
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		log.Fatalf("Error listening for UDP response: %v", err)
	}
	defer conn.Close()

	// Broadcast to 255.255.255.255 (Global Broadcast)
	broadcastAddr, err := net.ResolveUDPAddr("udp4", "255.255.255.255:9999")
	if err != nil {
		log.Fatalf("Error resolving broadcast address: %v", err)
	}

	msg := []byte("DISCOVER_GOPHER_FS")
	_, err = conn.WriteTo(msg, broadcastAddr)
	if err != nil {
		// Fallback: Try localhost if broadcast fails (useful for local testing/restrictions)
		log.Printf("Broadcast failed (%v), trying localhost...", err)
		localAddr, _ := net.ResolveUDPAddr("udp4", "127.0.0.1:9999")
		_, err = conn.WriteTo(msg, localAddr)
		if err != nil {
			log.Fatalf("Error communicating with server: %v", err)
		}
	}

	// Wait for response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 1024)
	
	n, remoteAddr, err := conn.ReadFrom(buf)
	if err != nil {
		log.Printf("Discovery timed out or failed: %v", err)
		return ""
	}
	
	tcpPort := string(buf[:n])
	
	// remoteAddr is an interface (net.Addr), we need the IP
	udpAddr, ok := remoteAddr.(*net.UDPAddr)
	if !ok {
		log.Printf("Could not get UDP address from response")
		return ""
	}
	
	serverIP := udpAddr.IP.String()
	fullAddr := serverIP + tcpPort
	fmt.Printf("Found server at %s\n", fullAddr)
	return fullAddr
}

func downloadFile(serverAddr, filename string) {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Fatalf("Error connecting to server: %v", err)
	}
	defer conn.Close()

	// Send file name length
	nameBytes := []byte(filename)
	nameLen := uint32(len(nameBytes))
	
	if err := binary.Write(conn, binary.LittleEndian, nameLen); err != nil {
		log.Fatalf("Error sending file name length: %v", err)
	}

	// Send file name
	if _, err := conn.Write(nameBytes); err != nil {
		log.Fatalf("Error sending file name: %v", err)
	}

	// Create local file
	outFile, err := os.Create("downloaded_" + filepath.Base(filename))
	if err != nil {
		log.Fatalf("Error creating local file: %v", err)
	}
	defer outFile.Close()

	// Stream from connection to file
	receivedBytes, err := io.Copy(outFile, conn)
	if err != nil {
		log.Fatalf("Error downloading file: %v", err)
	}

	fmt.Printf("Successfully downloaded %s (%d bytes)\n", filename, receivedBytes)
}
