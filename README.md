# MTR Tool

A versatile tool that provides both a REST API and CLI interface for running MTR (My TraceRoute) commands.

## Features

- Dual interface:
  - REST API for server mode
  - CLI for direct command-line usage
- Asynchronous execution in server mode
- Configurable packet count and report mode
- Color-coded output highlighting:
  - Red for high packet loss (≥10%)
  - Yellow for high latency (≥100ms)
- Input validation and security checks
- Detailed error reporting

## Requirements

- Go 1.21 or higher
- MTR command-line tool installed on the system
- Root privileges (sudo access)

## Environment Variables

The following environment variables can be used to customize the tool's behavior:

- `MTR_PATH`: Path to the MTR executable
  - Default on macOS (Homebrew): `/opt/homebrew/sbin/mtr`
  - Default in Docker: `/usr/sbin/mtr`
  - Example: `MTR_PATH=/usr/local/bin/mtr ./mtr-tool -host=google.com`

## Installation

### Local Development

1. Install dependencies:
   ```bash
   go mod download
   ```

2. Build the application:
   ```bash
   go build -o mtr-tool
   ```

## Usage

### CLI Mode

Run MTR directly from the command line:

```bash
sudo ./mtr-tool -host=google.com -count=10 -report=true
```

Options:
- `-host`: Target hostname or IP (required)
- `-count`: Number of packets to send (default: 20, max: 100)
- `-report`: Enable report mode (default: false)

### Server Mode

Run as an HTTP server:

```bash
sudo ./mtr-tool -server -port=8080
```

Options:
- `-server`: Enable server mode
- `-port`: Server port (default: 8080)

#### API Endpoint: GET /mtr

Parameters:
- `hostname` (required): The target hostname or IP address
- `count` (optional): Number of packets to send (default: 20, max: 100)
- `report` (optional): Enable report mode (default: false)

Example:
```bash
curl "http://localhost:8080/mtr?hostname=google.com&count=50&report=true"
```

Response format:
```json
{
  "status": "accepted",
  "message": "MTR trace to google.com started (count=50, report=true)"
}
```

The actual MTR output will be displayed in the server's console.

### Docker

1. Build the Docker image:
   ```bash
   docker build -t mtr-tool .
   ```

2. Run in CLI mode:
   ```bash
   # For accurate network metrics, use host network mode:
   docker run --network host --cap-add=NET_RAW --cap-add=NET_ADMIN mtr-tool -host=google.com -count=10

   # Or with default bridge network (note: first hop will be Docker's bridge):
   docker run --cap-add=NET_RAW --cap-add=NET_ADMIN mtr-tool -host=google.com -count=10
   ```

3. Run in server mode:
   ```bash
   # For accurate network metrics (port 8080 on host):
   docker run --network host --cap-add=NET_RAW --cap-add=NET_ADMIN mtr-tool -server

   # Or with bridge network and port mapping:
   docker run --cap-add=NET_RAW --cap-add=NET_ADMIN -p 8080:8080 mtr-tool -server
   ```

Note: 
- When using `--network host`, do not use `-p` port mapping as it's not compatible with host network mode
- When using the default bridge network, the first hop will be Docker's network bridge (usually 172.17.0.1)
- The `--cap-add=NET_RAW --cap-add=NET_ADMIN` capabilities are required for MTR to function properly

## Error Handling

The API returns appropriate HTTP status codes and error messages in JSON format:

```json
{
  "status": "error",
  "message": "Error message here"
}
```

Common error scenarios:
- Missing or invalid hostname
- Count exceeds maximum limit (100)
- Invalid parameter values
- MTR execution failures

## Security Notes

- The application requires root privileges to run MTR
- Input validation is performed on all parameters
- Hostname is checked for potentially dangerous characters
- Maximum count limit prevents resource exhaustion

## License

[Add your license here]
