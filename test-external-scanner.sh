#!/bin/bash

# Test script for external scanner configuration

echo "=== FinGuard External Scanner Test ==="
echo ""

# Build the Docker image
echo "1. Building Docker image..."
docker build -t finguard:latest . || exit 1
echo "✓ Image built successfully"
echo ""

# Test 1: Run with cloud mode (default)
echo "2. Testing Cloud Mode..."
echo "   Starting container with FSS_API_KEY..."
docker run -d \
  --name finguard-cloud \
  -p 3000:3000 \
  -p 3443:3443 \
  -e FSS_API_KEY=test-key-here \
  finguard:latest

echo "   Waiting for container to start..."
sleep 5

echo "   Checking logs..."
docker logs finguard-cloud 2>&1 | grep -i "scanner\|mode" | head -5

echo "   Stopping container..."
docker stop finguard-cloud > /dev/null 2>&1
docker rm finguard-cloud > /dev/null 2>&1
echo "✓ Cloud mode test complete"
echo ""

# Test 2: Run with external mode
echo "3. Testing External Mode..."
echo "   Starting container with SCANNER_EXTERNAL_ADDR..."
docker run -d \
  --name finguard-external \
  -p 3000:3000 \
  -p 3443:3443 \
  -e SCANNER_EXTERNAL_ADDR=10.10.21.201:50051 \
  -e SCANNER_USE_TLS=false \
  finguard:latest

echo "   Waiting for container to start..."
sleep 5

echo "   Checking logs..."
docker logs finguard-external 2>&1 | grep -i "scanner\|mode\|external" | head -10

echo ""
echo "=== Test Summary ==="
echo "Container is running at:"
echo "  HTTP:  http://localhost:3000"
echo "  HTTPS: https://localhost:3443"
echo ""
echo "Login credentials:"
echo "  Admin: admin / admin123"
echo "  User:  user / user123"
echo ""
echo "To view logs: docker logs -f finguard-external"
echo "To stop: docker stop finguard-external && docker rm finguard-external"
echo ""
echo "Test the external scanner by:"
echo "  1. Log in as admin"
echo "  2. Go to Configuration page"
echo "  3. Verify External Scanner Address shows: 10.10.21.201:50051"
echo "  4. Upload a test file on Dashboard"
echo ""
