package discovery

import (
	"fmt"
	"log"
	"net"
	"time"
)

const DiscoveryPort = 9999
const DiscoveryMsg = "DISCOVER_GOPHER_FS"

// Listen listens for UDP broadcasts and responds with the server's TCP port
func Listen(serviceTCPPort string) {
	addr := &net.UDPAddr{
		Port: DiscoveryPort,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		log.Printf("Warning: UDP Discovery disabled (Error binding %d: %v)", DiscoveryPort, err)
		return
	}
	defer conn.Close()

	fmt.Printf("Discovery Server listening on UDP %d\n", DiscoveryPort)

	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error reading UDP: %v", err)
			continue
		}
		
		msg := string(buf[:n])
		if msg == DiscoveryMsg {
			log.Printf("Received discovery request from %s", remoteAddr)
			// Respond with our TCP port
			_, err := conn.WriteToUDP([]byte(serviceTCPPort), remoteAddr)
			if err != nil {
				log.Printf("Error sending discovery response: %v", err)
			}
		}
	}
}

// FindServer broadcasts a discovery message and returns the server's TCP address
func FindServer() string {
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

	msg := []byte(DiscoveryMsg)
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
