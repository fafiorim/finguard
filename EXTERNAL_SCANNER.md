# External Scanner Configuration

This guide explains how to configure FinGuard to use an external Vision One File Security scanner (e.g., Kubernetes deployment) instead of the cloud-based AMaaS API.

## Overview

FinGuard supports two scanner modes:

1. **Cloud Mode** (default): Uses Trend Micro Cloud One File Security API
   - Requires: `FSS_API_KEY`
   - Protocol: HTTP REST API
   - Endpoint: `antimalware.us-1.cloudone.trendmicro.com:443`

2. **External Mode**: Uses on-premise Vision One File Security scanner
   - Requires: `SCANNER_EXTERNAL_ADDR` (host:port)
   - Protocol: gRPC
   - No API key needed
   - Example: Kubernetes Vision One File Security

## Configuration via Web UI

1. Log in as administrator
2. Navigate to **Configuration** page
3. Scroll to **Scanner Configuration** section
4. Configure **External gRPC Scanner (Optional)**:
   - **Address**: Enter host:port (e.g., `10.10.21.201:50051`)
   - **Use TLS**: Check if the external scanner requires TLS connection
   - **Test Connection**: Click to verify connectivity before saving
5. Click **Save Configuration**
6. The scanner will automatically restart with the new configuration

### Testing Scanner Connection

Before saving your configuration, you can test the connection to the external scanner:

1. Enter the scanner address (e.g., `10.10.21.201:50051`)
2. Select TLS option if required
3. Click **Test Connection** button
4. Wait for the result:
   - ✓ **Success**: Green checkmark indicates the scanner is reachable
   - ✗ **Failed**: Red X with error details (check network/firewall)
   - ⚠ **Warning**: Yellow icon if address format is invalid

This helps verify your scanner is accessible before applying the configuration.

## Configuration via Environment Variables

When starting the Docker container, you can set:

```bash
# For external scanner mode:
docker run -d \
  -p 3000:3000 \
  -p 3443:3443 \
  -e SCANNER_EXTERNAL_ADDR=10.10.21.201:50051 \
  -e SCANNER_USE_TLS=false \
  finguard:latest

# For cloud mode (default):
docker run -d \
  -p 3000:3000 \
  -p 3443:3443 \
  -e FSS_API_KEY=your-api-key-here \
  finguard:latest
```

## How It Works

### Cloud Mode
```
FinGuard → HTTP REST API → Cloud One File Security
```

### External Mode
```
FinGuard → gRPC → Kubernetes Vision One → File Scanning
```

## External Scanner Details

For Kubernetes Vision One File Security:

- **gRPC Port**: 50051
- **HTTP Port**: 3001 (for health checks)
- **Admin Port**: 1344
- **Protocol**: gRPC (not HTTP REST)
- **TLS**: Optional (depends on deployment)

## Switching Between Modes

The scanner mode is determined by:

1. If `SCANNER_EXTERNAL_ADDR` is set → External Mode
2. If `FSS_API_KEY` is set → Cloud Mode
3. Priority: External mode takes precedence if both are set

## Troubleshooting

### Error: "Parse Error: Expected HTTP/"
- **Cause**: Trying to connect to a gRPC scanner using HTTP REST
- **Solution**: Use the External Scanner configuration with proper host:port

### Scanner Not Connecting
- Check network connectivity to the external scanner
- Verify the port is correct (50051 for gRPC, not 3001)
- Ensure firewall rules allow the connection
- Try with/without TLS option

### API Key Error When Using External Scanner
- **Cause**: `FSS_API_KEY` is still set in environment
- **Solution**: Remove or unset `FSS_API_KEY` when using external scanner

## Example: Connecting to Kubernetes Vision One

```yaml
# Docker Compose
services:
  finguard:
    image: finguard:latest
    ports:
      - "3000:3000"
      - "3443:3443"
    environment:
      - SCANNER_EXTERNAL_ADDR=10.10.21.201:50051
      - SCANNER_USE_TLS=false
      - ADMIN_USERNAME=admin
      - ADMIN_PASSWORD=secure_password
```

## Scanner Restart

When you change the external scanner configuration via the Web UI:

1. The configuration is saved to the system
2. The scanner process is automatically restarted
3. The new configuration takes effect immediately
4. No container restart is required

## Verification

To verify the scanner is using external mode, check the logs:

```bash
docker logs <container_id> | grep -i scanner
```

You should see:
```
Scanner started in External Scanner mode
External Scanner Address: 10.10.21.201:50051
TLS Enabled: false
```

For cloud mode:
```
Scanner started in Cloud Scanner mode
Region: us-1
```
