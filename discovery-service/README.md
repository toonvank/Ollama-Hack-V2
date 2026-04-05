# Discovery Service

Standalone microservice for discovering Ollama endpoints across IP ranges.

## Features

- 🔍 **Autonomous Scanning**: Periodically scans configured IP ranges
- 🚀 **High Performance**: 100 concurrent workers by default
- 🎯 **Precise Detection**: Verifies Ollama by checking for "Ollama is running"
- 📡 **API Integration**: Automatically sends discovered endpoints to main backend
- 🌐 **HTTP API**: Trigger manual scans and check status
- 🐳 **Docker Ready**: Easy deployment with Docker Compose

## Quick Start

### With Docker Compose

```bash
# Already configured in main docker-compose.yml
docker-compose up discovery-service
```

### Standalone

```bash
cd discovery-service
go build
./discovery-service
```

## Configuration

Environment variables (see `.env.example`):

```bash
# How often to scan (in hours)
SCAN_INTERVAL_HOURS=24

# IP ranges to scan
SCAN_IP_RANGES=192.168.1.0/24,10.0.0.0/16

# Main backend URL
BACKEND_URL=http://backend-go:8000

# API key for backend authentication
BACKEND_API_KEY=your_jwt_token_here

# Concurrent workers
MAX_WORKERS=100

# Service port
PORT=8001
```

## API Endpoints

### Health Check
```bash
GET /health
```

### Trigger Manual Scan
```bash
POST /scan
Content-Type: application/json

{
  "ip_range": "192.168.1.0/24"
}
```

### Get Status
```bash
GET /status
```

## Usage Examples

### Trigger a scan from command line:
```bash
curl -X POST http://localhost:8001/scan \
  -H "Content-Type: application/json" \
  -d '{"ip_range": "192.168.1.0/24"}'
```

### Check service status:
```bash
curl http://localhost:8001/status
```

### Health check:
```bash
curl http://localhost:8001/health
```

## How It Works

1. **Periodic Scanning**: Scans configured IP ranges every N hours
2. **Port Detection**: Checks if port 11434 is open (TCP)
3. **Verification**: Confirms Ollama by checking HTTP response
4. **Batch Import**: Sends discovered endpoints to main backend
5. **Auto-Testing**: Backend automatically tests new endpoints

## Performance

- **Small network** (192.168.1.0/24): ~5-10 seconds
- **Medium network** (10.0.0.0/16): ~10-30 minutes  
- **100 concurrent workers** scanning in parallel
- **2-second timeout** per port scan
- **5-second timeout** per HTTP verification

## Integration

The service integrates with the main backend via:

- **POST /api/v2/endpoint/batch**: Sends discovered endpoints
- **Authorization header**: Uses JWT token or API key
- **Automatic deduplication**: Backend handles duplicate prevention

## Logs

The service provides detailed logging:

```
🚀 Discovery Service starting...
   Backend URL: http://backend-go:8000
   Max Workers: 100
   Scan Interval: 24h0m0s
   Default IP Ranges: [192.168.1.0/24]
✅ Discovery Service started successfully
🌐 HTTP API listening on port 8001
⏰ Triggering scheduled scan
🔍 Processing scan from queue: 192.168.1.0/24
🔎 Starting scan of 192.168.1.0/24
   Expanded to 254 IP addresses
   ✓ Found Ollama at http://192.168.1.100:11434
✅ Sent 3 endpoints to backend
✅ Scan completed: 192.168.1.0/24 - found 3 endpoints in 8s
```

## Security Notes

⚠️ **Important**: Only scan networks you own or have permission to scan.

- Unauthorized scanning may violate laws
- Some networks have intrusion detection
- Be respectful of network resources
- Use rate limiting appropriately

## Architecture

```
┌──────────────────────┐
│  Discovery Service   │
│     (Port 8001)      │
└──────────┬───────────┘
           │
           ├─ Periodic Scanner (24h)
           ├─ Manual Scan API
           ├─ Status API
           │
           ▼
    ┌─────────────────┐
    │   Worker Pool   │
    │  (100 workers)  │
    └────────┬────────┘
             │
             ├─ TCP:11434 Check
             ├─ HTTP Verification
             └─ Batch Send
                    │
                    ▼
           ┌────────────────────┐
           │  Main Backend API  │
           │  POST /endpoint/batch │
           └────────────────────┘
```

## Development

Build:
```bash
go build -o discovery-service
```

Run locally:
```bash
export SCAN_IP_RANGES="192.168.1.0/24"
export BACKEND_URL="http://localhost:8000"
./discovery-service
```

Build Docker image:
```bash
docker build -t discovery-service .
```

## License

Same as main Ollama-Hack project.
