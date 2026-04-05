# Discovery Service Setup Guide

## What is This?

The Discovery Service is a **standalone microservice** that automatically finds Ollama endpoints on your network. Instead of paying for FOFA or other services, this scans your specified IP ranges and discovers running Ollama instances.

## Features ✨

- 🔍 **Automatic Discovery**: Scans IP ranges every 24 hours
- ⚡ **Fast**: Uses 100 concurrent workers
- 🎯 **Precise**: Verifies endpoints by checking "Ollama is running"  
- 📡 **Auto-Import**: Sends discovered endpoints to main backend
- 🌐 **API**: Trigger manual scans via HTTP API
- 🐳 **Containerized**: Runs as independent Docker service

## Quick Start

### 1. Configure IP Ranges

Edit your `.env` file or set environment variables:

```bash
# Scan your local network
export SCAN_IP_RANGES="192.168.1.0/24"

# Or scan multiple networks
export SCAN_IP_RANGES="192.168.1.0/24,10.0.0.0/24,172.16.0.0/16"
```

### 2. Get Backend API Key

You need an admin JWT token for the discovery service to authenticate with the backend:

```bash
# Login to get JWT token
curl -X POST http://localhost:8000/api/v2/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your_password"}'

# Copy the "access_token" from response
export BACKEND_API_KEY="your_jwt_token_here"
```

### 3. Start the Service

```bash
# With Docker Compose (recommended)
docker-compose up -d discovery-service

# Or standalone
cd discovery-service
go build
./discovery-service
```

### 4. Monitor Logs

```bash
docker-compose logs -f discovery-service
```

You should see:
```
🚀 Discovery Service starting...
   Backend URL: http://backend:8000
   Max Workers: 100
   Scan Interval: 24h0m0s
   Default IP Ranges: [192.168.1.0/24]
✅ Discovery Service started successfully
🌐 HTTP API listening on port 8001
```

## Configuration

### Environment Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `SCAN_IP_RANGES` | IP ranges to scan (comma-separated CIDR) | _(empty)_ | `192.168.1.0/24,10.0.0.0/16` |
| `SCAN_INTERVAL_HOURS` | How often to scan | `24` | `12` |
| `BACKEND_URL` | Main application URL | `http://backend:8000` | `http://localhost:8000` |
| `BACKEND_API_KEY` | JWT token for auth | _(empty)_ | `eyJhbGc...` |
| `MAX_WORKERS` | Concurrent scan workers | `100` | `200` |
| `PORT` | HTTP API port | `8001` | `8080` |

### Full .env Example

Create a `.env` file in the root directory:

```bash
# Discovery Service Config
SCAN_IP_RANGES=192.168.1.0/24,10.0.0.0/24
SCAN_INTERVAL_HOURS=12
BACKEND_API_KEY=your_jwt_token_from_login
MAX_WORKERS=100
```

## Usage

### Manual Scan via API

Trigger a one-time scan:

```bash
curl -X POST http://localhost:8001/scan \
  -H "Content-Type: application/json" \
  -d '{"ip_range": "192.168.50.0/24"}'
```

Response:
```json
{
  "status": "queued",
  "ip_range": "192.168.50.0/24",
  "message": "Scan queued successfully",
  "started_at": "2024-04-05T15:30:00Z"
}
```

### Check Scanner Status

```bash
curl http://localhost:8001/status
```

Response:
```json
{
  "active_scans": {
    "192.168.1.0/24": {
      "IPRange": "192.168.1.0/24",
      "Status": "running",
      "Discovered": 3,
      "TotalIPs": 254,
      "Scanned": 180,
      "StartedAt": "2024-04-05T15:30:00Z"
    }
  },
  "queue_size": 0
}
```

### Health Check

```bash
curl http://localhost:8001/health
```

## How It Works

```
┌──────────────────────────────────────────────────────┐
│                 Discovery Service                     │
│                    (Port 8001)                        │
└───────────────────┬──────────────────────────────────┘
                    │
         ┌──────────┴──────────┐
         │                     │
    ┌────▼────┐         ┌──────▼──────┐
    │ Timer   │         │  HTTP API   │
    │ (24h)   │         │  /scan      │
    └────┬────┘         └──────┬──────┘
         │                     │
         └──────────┬──────────┘
                    │
             ┌──────▼──────┐
             │ Scan Queue  │
             └──────┬──────┘
                    │
         ┌──────────▼──────────┐
         │   Worker Pool       │
         │   (100 workers)     │
         └──────────┬──────────┘
                    │
         ┌──────────▼──────────┐
         │  For each IP:       │
         │  1. TCP port check  │
         │  2. HTTP verify     │
         │  3. Check "Ollama"  │
         └──────────┬──────────┘
                    │
         ┌──────────▼──────────────┐
         │  Batch Send to Backend  │
         │  POST /endpoint/batch   │
         └─────────────────────────┘
```

## Performance Tips

### Network Size vs Time

| Network Size | IPs | Estimated Time |
|--------------|-----|----------------|
| /32 (single) | 1 | < 1 second |
| /24 (small) | 254 | 5-10 seconds |
| /20 (medium) | 4,094 | 1-2 minutes |
| /16 (large) | 65,534 | 10-30 minutes |

### Optimization

**For faster scans:**
```bash
MAX_WORKERS=200  # More concurrent workers
```

**For slower/safer scans:**
```bash
MAX_WORKERS=50   # Fewer workers = less aggressive
```

**Adjust scan frequency:**
```bash
SCAN_INTERVAL_HOURS=6   # Scan every 6 hours
SCAN_INTERVAL_HOURS=168 # Scan once per week
```

## Troubleshooting

### No Endpoints Found

**Check logs:**
```bash
docker-compose logs discovery-service | grep "Found Ollama"
```

**Verify Ollama is accessible:**
```bash
curl http://192.168.1.100:11434
# Should return: "Ollama is running"
```

**Check firewall:**
```bash
# Make sure port 11434 is accessible
telnet 192.168.1.100 11434
```

### Authentication Errors

If you see `backend returned status 401`:

1. Get a fresh JWT token:
```bash
curl -X POST http://localhost:8000/api/v2/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your_password"}'
```

2. Update `BACKEND_API_KEY` in .env or docker-compose.yml

3. Restart service:
```bash
docker-compose restart discovery-service
```

### Service Not Starting

**Check Docker logs:**
```bash
docker-compose logs discovery-service
```

**Verify backend is running:**
```bash
curl http://localhost:8000/health
```

**Check IP range format:**
```bash
# Valid formats:
SCAN_IP_RANGES=192.168.1.0/24         # Single network
SCAN_IP_RANGES=192.168.1.100          # Single IP
SCAN_IP_RANGES=192.168.1.0/24,10.0.0.0/16  # Multiple networks

# Invalid:
SCAN_IP_RANGES=192.168.1.*            # No wildcards
SCAN_IP_RANGES=192.168.1.0-255        # No ranges
```

## Security Considerations

### ⚠️ Important

**Only scan networks you own or have permission to scan!**

- Unauthorized network scanning may be illegal
- Some networks have intrusion detection systems
- You could be blocked or flagged

### Best Practices

1. **Start small**: Test with /24 networks first
2. **Off-peak hours**: Schedule scans when traffic is low  
3. **Whitelist**: Only scan known/approved networks
4. **Monitor**: Watch logs for errors or issues
5. **Rate limit**: Don't set MAX_WORKERS too high

### Recommended Setup

For production:
```bash
# Conservative settings
SCAN_INTERVAL_HOURS=24
MAX_WORKERS=50
SCAN_IP_RANGES=192.168.1.0/24  # Only your local network
```

For testing/development:
```bash
# Aggressive settings
SCAN_INTERVAL_HOURS=1
MAX_WORKERS=200
SCAN_IP_RANGES=192.168.1.0/24,192.168.2.0/24
```

## Integration with Main App

### Automatic Flow

1. Discovery service scans IP ranges
2. Finds Ollama instances (port 11434 + "Ollama is running")
3. Batches discovered endpoints
4. POSTs to `http://backend:8000/api/v2/endpoint/batch`
5. Backend adds endpoints to database
6. Backend schedules availability tests
7. Endpoints appear in admin dashboard

### Manual Workflow

You can also manually trigger discovery from the main app (future feature):

```bash
# Future: Trigger from main backend
curl -X POST http://localhost:8000/api/v2/discovery/scan \
  -H "Authorization: Bearer YOUR_JWT" \
  -d '{"ip_range": "192.168.1.0/24"}'
```

## Monitoring

### View Active Scans

```bash
curl http://localhost:8001/status | jq
```

### Watch Logs in Real-Time

```bash
docker-compose logs -f discovery-service
```

### Check Discovered Endpoints

```bash
# In main backend
curl http://localhost:8000/api/v2/endpoint \
  -H "Authorization: Bearer YOUR_JWT" \
  | jq '.data[] | select(.name | startswith("Discovered"))'
```

## Advanced Usage

### Custom Port Scanning

Edit `discovery-service/main.go`:

```go
// Change from port 11434 to custom port
if ds.scanHost(ip, 8080) {  // Scan port 8080 instead
    discoveredChan <- ip
}
```

### Multiple Port Support

```go
ports := []int{11434, 11435, 11436}
for _, port := range ports {
    if ds.scanHost(ip, port) {
        endpoint := DiscoveredEndpoint{
            URL: fmt.Sprintf("http://%s:%d", ip, port),
            // ...
        }
        discovered = append(discovered, endpoint)
    }
}
```

### Custom Verification Logic

```go
// In scanHost function, change verification:
return strings.Contains(string(body), "Your Custom String")
```

## Next Steps

1. ✅ Configure IP ranges in `.env`
2. ✅ Get admin JWT token
3. ✅ Start discovery service
4. ✅ Monitor logs for discoveries
5. ✅ Check main dashboard for new endpoints
6. ✅ Test discovered endpoints

## FAQ

**Q: How much does this cost?**  
A: Free! No FOFA subscription needed.

**Q: Can I scan the entire internet?**  
A: Technically yes, but **don't**. It's unethical and likely illegal.

**Q: How many IPs can it scan?**  
A: Tested up to /16 (65k IPs). Larger ranges will take hours.

**Q: Does it work with IPv6?**  
A: Not yet. IPv6 support planned for future release.

**Q: Can I run multiple instances?**  
A: Yes, just use different IP ranges per instance.

**Q: Will it find password-protected Ollama instances?**  
A: No, only publicly accessible instances on port 11434.

## Support

For issues or questions:
1. Check logs: `docker-compose logs discovery-service`
2. Review this guide
3. Open an issue on GitHub

## License

Same as main Ollama-Hack project.
