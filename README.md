# Gopher-FS: Secure Distributed File Transfer System

A robust, concurrent, and secure file transfer system built in Go. This tool allows for decentralized server discovery, secure file uploads/downloads, and automatic data integrity verification.

## 🚀 Key Features

*   **Zero-Config Discovery**: Servers are automatically discovered on the local network using UDP broadcasting. No IP configuration needed.
*   **Secure Transport**: All file transfers are encrypted using TLS 1.3 (Self-Signed Certificates generated on-the-fly for this demo).
*   **Data Integrity**: Every file transfer is verified with SHA-256 checksums to ensure zero corruption.
*   **High Performance**: Uses `io.Copy` and Go's streaming interfaces to handle large files with minimal memory footprint.
*   **Concurrency**: Capable of handling multiple client requests simultaneously using lightweight Goroutines.

## 🛠 Architecture

The project is structured following standard Golang layout patterns:

*   `cmd/server`: The server application entry point. Handles TCP/TLS listening and concurrent client dispatch.
*   `cmd/client`: The client CLI tool. Handles discovery, connection, and file operations.
*   `internal/discovery`: UDP Multicast/Broadcast logic for service discovery.
*   `internal/protocol`: Defined binary protocol for efficient framing (Size, Name, Checksum, Data) and Operation Codes.
*   `internal/security`: Logic for ephemeral TLS certificate generation.

## 📦 Installation & Usage

**Prerequisites:** [Go 1.21+](https://go.dev/dl/)

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/yourusername/gopher-fs.git
    cd gopher-fs
    ```

2.  **Start the Server (Terminal 1):**
    ```bash
    go run cmd/server/main.go
    ```
    *Output:* `Secure File Server listening on :9000 (TLS enabled)`

3.  **Run the Client (Terminal 2):**

    *   **Download a File:**
        ```bash
        go run cmd/client/main.go -file my_document.txt
        ```

    *   **Upload a File:**
        ```bash
        go run cmd/client/main.go -file my_upload.png -upload
        ```

## 🔒 Security & Protocol Detail

### Binary Protocol
| Size (Bytes) | Field | Description |
| :--- | :--- | :--- |
| 1 | OpCode | `0x01` (Download) or `0x02` (Upload) |
| 4 | NameLen | Length of the filename |
| 8 | FileSize | Size of the file in bytes |
| 32 | Checksum | SHA-256 Hash of the file |
| N | Name | The filename string |
| M | Data | Raw file content stream |

### Encryption
All TCP connections are upgraded to TLS automatically using ephemeral keys. This prevents passive network sniffing from reading your files.

## 📝 License
MIT License
