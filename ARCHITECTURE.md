# FinGuard Scanner Architecture

## Overview

FinGuard uses a **3-tier architecture** for file scanning:

```
┌─────────────────┐
│   Web App       │  (Node.js - port 3000/3443)
│   (server.js)   │  • Upload interface
└────────┬────────┘  • Configuration UI
         │           • Scan history
         │ HTTP
         ↓
┌─────────────────┐
│ Scanner Service │  (Go - port 3001)
│  (scanner.go)   │  • HTTP API wrapper
└────────┬────────┘  • SDK integration
         │ SDK
         ↓
┌─────────────────┐
│ Scanning Engine │
│                 │  Cloud API -OR- External gRPC
└─────────────────┘
```

## Components

### 1. Web Application (Node.js)
**File:** `server.js`  
**Port:** 3000 (HTTP), 3443 (HTTPS)  
**Purpose:** User interface and file management

- Handles file uploads
- Manages configuration
- Displays scan results
- Communicates with scanner service at `http://localhost:3001`

### 2. Scanner Service (Go)
**File:** `scanner.go`  
**Port:** 3001 (HTTP)  
**Purpose:** Bridge between web app and scanning engines

- Provides HTTP REST API for the web app
- Integrates with Trend Micro SDK
- Supports two backend modes:
  - **Cloud Mode**: Connects to TrendAI File Security API
  - **External Mode**: Connects to on-premise gRPC scanner

### 3. Scanning Engine
**Purpose:** Actual malware detection

**Option A: Cloud API** (default)
- Requires: `FSS_API_KEY`
- Protocol: HTTPS
- Endpoint: TrendAI File Security service

**Option B: External gRPC**
- Requires: `SCANNER_EXTERNAL_ADDR`
- Protocol: gRPC (optional TLS)
- Example: Kubernetes Vision One at 10.10.21.201:50051

## Configuration Modes

### Mode 1: Cloud Scanner (Default)

```bash
docker run -d \
  -p 3000:3000 \
  -p 3443:3443 \
  -e FSS_API_KEY=your-api-key \
  finguard:latest
```

**Flow:**
```
User → Web App (3000) → Scanner Service (3001) → Cloud API → Scan
```

**Settings:**
- Scanner Service URL: `http://localhost:3001` (keep as is)
- External gRPC Scanner: (leave empty)

### Mode 2: External Scanner

```bash
docker run -d \
  -p 3000:3000 \
  -p 3443:3443 \
  -e SCANNER_EXTERNAL_ADDR=10.10.21.201:50051 \
  -e SCANNER_USE_TLS=false \
  finguard:latest
```

**Flow:**
```
User → Web App (3000) → Scanner Service (3001) → External gRPC → Scan
```

**Settings:**
- Scanner Service URL: `http://localhost:3001` (keep as is)
- External gRPC Scanner: `10.10.21.201:50051`

## Understanding the Settings

### Scanner Service URL
**What it is:** The internal HTTP endpoint where the web app talks to the scanner service  
**Default:** `http://localhost:3001`  
**When to change:** Only if you split the scanner service to a separate container

```
┌──────────────┐      ┌──────────────┐
│   Web App    │─────▶│   Scanner    │
│  Container A │ 3001 │  Container B │
└──────────────┘      └──────────────┘
```

In this case, use: `http://scanner-container:3001`

### External gRPC Scanner
**What it is:** The backend that the scanner service uses  
**Options:**
- **Empty (default):** Uses TrendAI File Security API
- **host:port:** Uses your own Vision One File Security deployment

```
Web App ───┐
           ├──▶ Scanner Service ───┐
Files ─────┘                       ├──▶ Cloud API (default)
                                   └──▶ External gRPC (if configured)
```

## Common Scenarios

### Scenario 1: All-in-One (Default)
Everything in one container, using cloud API:

```bash
docker run -d \
  -e FSS_API_KEY=xxx \
  -p 3000:3000 \
  finguard:latest
```

**Config UI:**
- Scanner Service URL: `http://localhost:3001` ✓
- External gRPC Scanner: (empty) ✓

### Scenario 2: All-in-One with External Scanner
Everything in one container, using Kubernetes scanner:

```bash
docker run -d \
  -e SCANNER_EXTERNAL_ADDR=10.10.21.201:50051 \
  -p 3000:3000 \
  finguard:latest
```

**Config UI:**
- Scanner Service URL: `http://localhost:3001` ✓
- External gRPC Scanner: `10.10.21.201:50051` ✓

### Scenario 3: Separate Scanner Service
Web app and scanner in different containers:

**Container 1 (Web App Only):**
```bash
docker run -d \
  --name finguard-web \
  -p 3000:3000 \
  finguard:webapp
```

**Container 2 (Scanner Only):**
```bash
docker run -d \
  --name finguard-scanner \
  -e FSS_API_KEY=xxx \
  finguard:scanner
```

**Config UI:**
- Scanner Service URL: `http://finguard-scanner:3001` ✓
- External gRPC Scanner: (empty) ✓

## How to Switch Modes

### Switching to External Scanner

1. **Stop the container**
2. **Start with external scanner environment variables:**
   ```bash
   docker run -d \
     -e SCANNER_EXTERNAL_ADDR=10.10.21.201:50051 \
     -e SCANNER_USE_TLS=false \
     finguard:latest
   ```
3. **Or use the Config UI** (requires container restart to apply)

### Switching to Cloud API

1. **Stop the container**
2. **Start with API key:**
   ```bash
   docker run -d \
     -e FSS_API_KEY=your-key \
     finguard:latest
   ```
3. **Remove external scanner setting from Config UI**

## Verification

### Check Scanner Mode

```bash
docker logs <container> | grep -i mode
```

**Cloud Mode:**
```
- Mode: Cloud Scanner (API)
- Region: us-1
```

**External Mode:**
```
- Mode: External Scanner (gRPC)
- Scanner Address: 10.10.21.201:50051
- TLS: false
```

### Test Connection

In the Config UI:
1. Enter external scanner address
2. Click "Test Connection"
3. Should show ✓ if reachable

## Troubleshooting

### "Scanner Service URL" not working
**Problem:** Changed from `localhost:3001` but can't connect  
**Solution:** Keep it as `localhost:3001` unless you split containers

### External scanner not working
**Problem:** Test connection fails  
**Causes:**
- Wrong IP address or port
- Network/firewall blocking connection
- External scanner not running
- Using HTTP port (3001) instead of gRPC port (50051)

**Fix:**
1. Verify scanner is at 10.10.21.201:50051
2. Check with: `telnet 10.10.21.201 50051`
3. Use gRPC port (50051), not HTTP port (3001/3443)

### Files not scanning
**Problem:** Upload works but no scan results  
**Check:**
1. Scanner service logs: `docker exec <container> cat /app/scanner.log`
2. Verify mode is correct (Cloud vs External)
3. If cloud mode: Check FSS_API_KEY is valid
4. If external mode: Verify gRPC scanner is reachable

## Summary

✅ **Keep "Scanner Service URL" as `http://localhost:3001`** unless you have a separate scanner container

✅ **Use "External gRPC Scanner"** to choose between:
- Empty = Cloud API (needs FSS_API_KEY)
- host:port = Your own scanner (no API key needed)

✅ **Restart container** after changing external scanner mode

✅ **Test connection** before uploading files
