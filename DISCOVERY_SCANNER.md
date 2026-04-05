# Ollama Endpoint Discovery Scanner

## Overview

The Discovery Scanner is a built-in microservice that automatically discovers Ollama endpoints across specified IP ranges. Instead of relying on third-party services like FOFA or Shodan, this scanner actively probes your network to find running Ollama instances.

## How It Works

The scanner identifies Ollama endpoints by:

1. **Port Scanning**: Checking if port 11434 (default Ollama port) is open
2. **Verification**: Making an HTTP request to confirm the response contains "Ollama is running"
3. **Auto-Import**: Automatically adding discovered endpoints to the database
4. **Auto-Test**: Scheduling immediate testing for newly discovered endpoints

## Configuration

### Environment Variables

Add these to your `.env` file:

```bash
# IP ranges to scan (CIDR notation, comma-separated)
SCAN_IP_RANGES=192.168.1.0/24,10.0.0.0/16

# Optional: Shodan API for passive discovery
SHODAN_API_KEY=your_shodan_api_key_here
```

### Scan Interval

By default, the scanner runs every 24 hours. You can modify this in the code:

```go
// In discovery_scanner.go
interval: 24 * time.Hour, // Adjust as needed
```

### Performance Tuning

You can adjust scanner performance by modifying these parameters in `discovery_scanner.go`:

```go
maxWorkers:   100,            // Concurrent scan workers (default: 100)
scanTimeout:  2 * time.Second, // Port scan timeout (default: 2s)
httpTimeout:  5 * time.Second, // HTTP request timeout (default: 5s)
```

## API Endpoints

### Trigger Manual Scan

**POST** `/api/v2/discovery/scan`

Trigger a manual scan of a specific IP range (admin only).

**Request:**
```json
{
  "ip_range": "192.168.1.0/24"
}
```

**Response:**
```json
{
  "message": "Manual scan triggered",
  "ip_range": "192.168.1.0/24"
}
```

**Example with curl:**
```bash
curl -X POST http://localhost:8000/api/v2/discovery/scan \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"ip_range": "192.168.1.0/24"}'
```

### Get Scanner Status

**GET** `/api/v2/discovery/status`

Get the current status of the discovery scanner (admin only).

**Response:**
```json
{
  "status": "running",
  "message": "Discovery scanner is active"
}
```

## Usage Examples

### Example 1: Scan Local Network

```bash
# Scan your local network (192.168.1.0/24)
SCAN_IP_RANGES=192.168.1.0/24
```

### Example 2: Scan Multiple Networks

```bash
# Scan multiple networks
SCAN_IP_RANGES=192.168.1.0/24,10.0.0.0/24,172.16.0.0/24
```

### Example 3: Scan Large Range

```bash
# Scan entire private network ranges (warning: this will take a long time!)
SCAN_IP_RANGES=10.0.0.0/16
```

### Example 4: Manual Scan via API

```bash
# Trigger a one-time scan of a specific range
curl -X POST http://localhost:8000/api/v2/discovery/scan \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"ip_range": "192.168.50.0/24"}'
```

## Scanner Behavior

### Automatic Features

1. **Duplicate Prevention**: Won't add endpoints that already exist in the database
2. **Auto-Naming**: Discovered endpoints are named "Discovered: [IP]"
3. **Auto-Testing**: New endpoints are immediately scheduled for availability testing
4. **Background Processing**: Scans run in the background without blocking the API

### Performance Characteristics

- **100 concurrent workers** by default
- **2-second timeout** per port scan
- **5-second timeout** per HTTP verification
- For a /24 network (254 IPs): ~5-10 seconds
- For a /16 network (65,534 IPs): ~10-30 minutes

### Logging

The scanner provides detailed logging:

```
[discovery-scanner] starting Ollama endpoint discovery scanner
[discovery-scanner] starting scan of 2 IP range(s)
[discovery-scanner] scanning range: 192.168.1.0/24
[discovery-scanner] expanded 192.168.1.0/24 to 254 IPs
[discovery-scanner] ✓ found Ollama at http://192.168.1.100:11434
[discovery-scanner] added new endpoint: http://192.168.1.100:11434 (ID: 42)
[discovery-scanner] discovered 3 new Ollama endpoints in 192.168.1.0/24
[discovery-scanner] scan complete
```

## Security Considerations

### Network Scanning Ethics

⚠️ **Important**: Only scan networks you own or have explicit permission to scan.

- Unauthorized network scanning may violate laws and terms of service
- Some networks may have intrusion detection systems
- Be respectful of network resources

### Recommended Practices

1. **Start Small**: Begin with small ranges (e.g., /24) to test
2. **Off-Peak Hours**: Schedule scans during low-traffic periods
3. **Rate Limiting**: The scanner includes built-in concurrency limits
4. **Monitoring**: Watch logs to ensure scans complete successfully

## Integration with Existing Features

### Endpoint Testing

Discovered endpoints are automatically:
- Added to the endpoint testing queue
- Monitored by the health tracker
- Available for use through the Ollama proxy

### Shodan Integration

The scanner works alongside Shodan-based discovery:
- Both can run simultaneously
- Shodan provides global discovery (passive)
- This scanner provides local network discovery (active)

## Troubleshooting

### No Endpoints Found

1. Verify SCAN_IP_RANGES is set correctly
2. Ensure Ollama instances are running on port 11434
3. Check firewall rules allow connections to port 11434
4. Review logs for connection errors

### Slow Scans

1. Reduce the number of concurrent workers
2. Increase timeouts for slow networks
3. Scan smaller IP ranges
4. Run manual scans during off-peak hours

### Permission Errors

1. Ensure you're using an admin account
2. Check JWT token is valid
3. Verify API authentication is working

## Advanced Configuration

### Custom Ports

To scan custom ports, modify `discovery_scanner.go`:

```go
// In scanIPRange function
if s.scanHost(ip, 11434) {  // Change port here
    resultsChan <- ip
}
```

### Custom Verification

To use different verification logic:

```go
// In scanHost function
if strings.Contains(bodyStr, "Ollama is running") {  // Change verification here
    log.Printf("[discovery-scanner] ✓ found Ollama at %s", url)
    return true
}
```

## Architecture

```
┌─────────────────────────┐
│   Discovery Scanner     │
│   (Background Service)  │
└───────────┬─────────────┘
            │
            ├─── Periodic Scans (24h interval)
            ├─── Manual API Triggers
            │
            ▼
    ┌───────────────────┐
    │   IP Range        │
    │   Expansion       │
    └─────────┬─────────┘
              │
              ▼
    ┌───────────────────┐
    │   Worker Pool     │
    │   (100 workers)   │
    └─────────┬─────────┘
              │
              ├─── Port Scan (tcp:11434)
              ├─── HTTP Verification
              └─── Database Import
                   │
                   ▼
          ┌────────────────────┐
          │  Endpoint Testing  │
          │  Queue             │
          └────────────────────┘
```

## Future Enhancements

Potential improvements:
- [ ] Web UI for managing scans
- [ ] Scan history and statistics
- [ ] Configurable scan schedules
- [ ] Support for custom port ranges
- [ ] IPv6 support
- [ ] Scan result notifications
- [ ] Bandwidth throttling
- [ ] Scan progress tracking

## Contributing

To contribute to the discovery scanner:

1. Test changes with small IP ranges first
2. Add unit tests for new features
3. Document configuration changes
4. Update this README with new features

## License

Same as the main Ollama-Hack project.
