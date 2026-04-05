# Discovery Service - Quick Reference

## 🚀 Quick Start

```bash
# 1. Set IP ranges to scan
export SCAN_IP_RANGES="192.168.1.0/24"

# 2. Get admin JWT token
curl -X POST http://localhost:8000/api/v2/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your_password"}' \
  | jq -r '.access_token'

# 3. Set token
export BACKEND_API_KEY="your_jwt_token"

# 4. Start service
docker-compose up -d discovery-service

# 5. Watch logs
docker-compose logs -f discovery-service
```

## 📡 API Endpoints

### Health Check
```bash
curl http://localhost:8001/health
```

### Manual Scan
```bash
curl -X POST http://localhost:8001/scan \
  -H "Content-Type: application/json" \
  -d '{"ip_range": "192.168.1.0/24"}'
```

### Check Status
```bash
curl http://localhost:8001/status | jq
```

## ⚙️ Configuration

| Variable | Default | Example |
|----------|---------|---------|
| `SCAN_IP_RANGES` | - | `192.168.1.0/24,10.0.0.0/16` |
| `SCAN_INTERVAL_HOURS` | `24` | `12` |
| `BACKEND_URL` | `http://backend:8000` | - |
| `BACKEND_API_KEY` | - | `eyJhbGc...` |
| `MAX_WORKERS` | `100` | `200` |

## 🔍 How It Works

1. **Scans** IP ranges every N hours (default: 24)
2. **Checks** if port 11434 is open (TCP)
3. **Verifies** HTTP response contains "Ollama is running"
4. **Sends** discovered endpoints to main backend
5. **Auto-tests** new endpoints in backend

## 📊 Logs

```
🚀 Discovery Service starting...
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

## 🐛 Troubleshooting

### No Endpoints Found
```bash
# Check if Ollama is accessible
curl http://192.168.1.100:11434
# Should return: "Ollama is running"

# Check logs
docker-compose logs discovery-service | grep "Found Ollama"
```

### Auth Errors (401)
```bash
# Get fresh token
curl -X POST http://localhost:8000/api/v2/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your_password"}' \
  | jq -r '.access_token'

# Update BACKEND_API_KEY and restart
docker-compose restart discovery-service
```

### Service Won't Start
```bash
# Check logs
docker-compose logs discovery-service

# Verify backend is up
curl http://localhost:8000/health

# Check IP range format (must be valid CIDR)
# ✅ Valid: 192.168.1.0/24
# ❌ Invalid: 192.168.1.*
```

## ⚡ Performance

| Network | IPs | Time (100 workers) |
|---------|-----|--------------------|
| /32 | 1 | < 1s |
| /24 | 254 | 5-10s |
| /20 | 4,094 | 1-2min |
| /16 | 65,534 | 10-30min |

## 🔐 Security

⚠️ **Only scan networks you own or have permission to scan!**

- Start with small ranges (/24)
- Schedule scans during off-peak hours
- Monitor for errors
- Don't scan the internet

## 📚 Full Documentation

- Setup Guide: `DISCOVERY_SETUP.md`
- Service README: `discovery-service/README.md`
- Scanner Implementation: `DISCOVERY_SCANNER.md`

## 🎯 Examples

### Scan Single IP
```bash
curl -X POST http://localhost:8001/scan \
  -H "Content-Type: application/json" \
  -d '{"ip_range": "192.168.1.100"}'
```

### Scan Small Network
```bash
curl -X POST http://localhost:8001/scan \
  -H "Content-Type: application/json" \
  -d '{"ip_range": "192.168.1.0/24"}'
```

### Scan Multiple Networks (in env)
```bash
SCAN_IP_RANGES=192.168.1.0/24,10.0.0.0/24,172.16.0.0/16
```

### Check Discovered Endpoints
```bash
# From main backend
curl http://localhost:8000/api/v2/endpoint \
  -H "Authorization: Bearer YOUR_JWT" \
  | jq '.data[] | select(.name | startswith("Discovered"))'
```

## 🏗️ Architecture

```
Discovery Service (8001)
    │
    ├─ Periodic Scanner (24h)
    ├─ Manual Scan API
    └─ Status API
         │
         ▼
    Worker Pool (100)
         │
         ├─ TCP:11434 Check
         ├─ HTTP Verify
         └─ Batch Import
              │
              ▼
    Main Backend (8000)
    POST /api/v2/endpoint/batch
```

## ✅ Checklist

- [ ] Set `SCAN_IP_RANGES` environment variable
- [ ] Get admin JWT token from backend login
- [ ] Set `BACKEND_API_KEY` environment variable  
- [ ] Start service: `docker-compose up -d discovery-service`
- [ ] Check logs: `docker-compose logs -f discovery-service`
- [ ] Verify in dashboard: New endpoints appear
- [ ] Test manually: `curl http://localhost:8001/scan`

## 💡 Tips

- Start with one small network (/24) to test
- Check logs regularly for discoveries
- Use `MAX_WORKERS=50` for safer/slower scans
- Use `MAX_WORKERS=200` for faster scans
- Set `SCAN_INTERVAL_HOURS=6` for more frequent scans
- Discovered endpoints auto-test in backend
- Duplicates are automatically filtered out

## 🤝 Contributing

1. Test with small ranges first
2. Add unit tests for new features
3. Update documentation
4. Submit PR with clear description

---

**Remember**: This replaces FOFA and saves you money! 💰
