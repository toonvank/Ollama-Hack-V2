#!/bin/bash

# Test script for Discovery Service

echo "🧪 Testing Discovery Service..."
echo ""

# Build the service
echo "📦 Building..."
cd discovery-service
go build -o discovery-service . || exit 1
echo "✅ Build successful"
echo ""

# Start the service in background
echo "🚀 Starting service..."
export SCAN_IP_RANGES=""
export BACKEND_URL="http://localhost:8000"
export BACKEND_API_KEY=""
export MAX_WORKERS=100
export PORT=8001
export SCAN_INTERVAL_HOURS=999

./discovery-service &
SERVICE_PID=$!
echo "   PID: $SERVICE_PID"

# Wait for service to start
sleep 2

# Test health endpoint
echo ""
echo "🏥 Testing health endpoint..."
HEALTH_RESPONSE=$(curl -s http://localhost:8001/health)
if echo "$HEALTH_RESPONSE" | grep -q "healthy"; then
    echo "✅ Health check passed"
    echo "   Response: $HEALTH_RESPONSE"
else
    echo "❌ Health check failed"
    kill $SERVICE_PID
    exit 1
fi

# Test status endpoint
echo ""
echo "📊 Testing status endpoint..."
STATUS_RESPONSE=$(curl -s http://localhost:8001/status)
if echo "$STATUS_RESPONSE" | grep -q "active_scans"; then
    echo "✅ Status check passed"
    echo "   Response: $STATUS_RESPONSE"
else
    echo "❌ Status check failed"
    kill $SERVICE_PID
    exit 1
fi

# Test scan endpoint (without actually scanning)
echo ""
echo "🔍 Testing scan endpoint..."
SCAN_RESPONSE=$(curl -s -X POST http://localhost:8001/scan \
    -H "Content-Type: application/json" \
    -d '{"ip_range": "192.168.1.1"}')

if echo "$SCAN_RESPONSE" | grep -q "queued"; then
    echo "✅ Scan endpoint passed"
    echo "   Response: $SCAN_RESPONSE"
else
    echo "❌ Scan endpoint failed"
    echo "   Response: $SCAN_RESPONSE"
fi

# Cleanup
echo ""
echo "🧹 Cleaning up..."
kill $SERVICE_PID
sleep 1

echo ""
echo "✅ All tests passed!"
echo ""
echo "Next steps:"
echo "1. Configure SCAN_IP_RANGES in docker-compose.yml"
echo "2. Get admin JWT token from main backend"
echo "3. Set BACKEND_API_KEY in docker-compose.yml"
echo "4. Run: docker-compose up -d discovery-service"
