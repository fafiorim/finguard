# External Scanner Implementation - Change Log

## Overview
This update adds support for connecting FinGuard to external Vision One File Security scanners (e.g., Kubernetes deployments) that use gRPC protocol, eliminating the need for Cloud One API keys.

## Changes Made

### 1. scanner.go (Go Scanner Service)
**File**: `scanner.go`
**Lines Modified**: 47-92

**Changes**:
- Added environment variable support for `SCANNER_EXTERNAL_ADDR` and `SCANNER_USE_TLS`
- Implemented dual-mode client initialization:
  - **External Mode**: Uses `NewClientInternal()` with gRPC address when `SCANNER_EXTERNAL_ADDR` is set
  - **Cloud Mode**: Uses `NewClient()` with API key for cloud-based scanning (default)
- Added logging to indicate which mode is being used
- No API key required when using external scanner

**Code**:
```go
externalAddr := os.Getenv("SCANNER_EXTERNAL_ADDR")
useTLS := os.Getenv("SCANNER_USE_TLS") == "true"

var client *amaas.Client
var err error

if externalAddr != "" {
    // External scanner mode (gRPC)
    log.Printf("Scanner started in External Scanner mode")
    log.Printf("External Scanner Address: %s", externalAddr)
    log.Printf("TLS Enabled: %v", useTLS)
    
    client, err = amaas.NewClientInternal("", externalAddr, useTLS, "")
} else {
    // Cloud scanner mode
    log.Printf("Scanner started in Cloud Scanner mode")
    log.Printf("Region: %s", region)
    
    client, err = amaas.NewClient(apiKey, region)
}
```

### 2. server.js (Node.js Backend)
**File**: `server.js`
**Multiple sections modified**

**Changes**:

#### System Configuration (Lines ~27-37)
- Added `externalScannerAddr` field (empty string default)
- Added `externalScannerTLS` field (false default)

```javascript
const systemConfig = {
    securityMode: 'logOnly',
    scanMethod: 'buffer',
    digestEnabled: true,
    scannerUrl: 'http://localhost:3001',
    externalScannerAddr: '',
    externalScannerTLS: false,
    // ... other fields
};
```

#### GET /api/config Endpoint
- Returns external scanner configuration to UI

#### POST /api/config Endpoint (Lines ~349-439)
- Made route handler async
- Extracts `externalScannerAddr` and `externalScannerTLS` from request body
- Validates and updates configuration
- Sets `needsScannerRestart` flag when external scanner settings change
- Implements scanner process restart logic:
  - Kills existing scanner process
  - Spawns new scanner with updated environment variables
  - Waits for scanner to initialize

```javascript
if (needsScannerRestart) {
    const { spawn } = require('child_process');
    
    // Kill existing scanner process
    try {
        require('child_process').execSync('pkill -f "./scanner"');
    } catch (e) {
        // Process might not be running
    }
    
    // Start new scanner with updated environment
    const scannerEnv = {
        ...process.env,
        SCANNER_EXTERNAL_ADDR: systemConfig.externalScannerAddr,
        SCANNER_USE_TLS: systemConfig.externalScannerTLS.toString()
    };
    
    const scanner = spawn('./scanner', [], {
        env: scannerEnv,
        detached: true,
        stdio: 'ignore'
    });
    scanner.unref();
    
    await new Promise(resolve => setTimeout(resolve, 2000));
}
```

### 3. configuration.html (Admin UI)
**File**: `public/configuration.html`
**Multiple sections modified**

**Changes**:

#### HTML UI Section (Lines ~285-308)
- Added "External gRPC Scanner (Optional)" configuration section
- Text input for host:port address (placeholder: `10.10.21.201:50051`)
- Checkbox for TLS option
- Description explaining Vision One File Security gRPC connection

```html
<div class="config-option">
    <div class="config-option-content">
        <div class="config-title">
            External gRPC Scanner (Optional)
            <span class="config-tag">Advanced</span>
        </div>
        <div class="config-description">
            Configure direct connection to Vision One File Security gRPC scanner...
        </div>
        <input type="text" name="externalScannerAddr" id="externalScannerAddr" 
               placeholder="10.10.21.201:50051 (leave empty for cloud)">
        <div>
            <label>
                <input type="checkbox" name="externalScannerTLS" id="externalScannerTLS">
                <span>Use TLS for external scanner connection</span>
            </label>
        </div>
    </div>
</div>
```

#### JavaScript loadConfig() Function (Lines ~449-451)
- Loads external scanner address from config
- Sets TLS checkbox state

```javascript
document.getElementById('externalScannerAddr').value = config.externalScannerAddr || '';
document.getElementById('externalScannerTLS').checked = config.externalScannerTLS === true;
```

#### JavaScript saveConfig() Function (Lines ~477-479, ~513-515)
- Extracts external scanner values from form
- Includes them in API request body

```javascript
const externalScannerAddr = document.getElementById('externalScannerAddr').value.trim();
const externalScannerTLS = document.getElementById('externalScannerTLS').checked;

// In fetch body:
body: JSON.stringify({ 
    // ... other fields
    externalScannerAddr,
    externalScannerTLS,
    // ... other fields
})
```

## New Features

### 1. Dual Scanner Mode Support
- **Cloud Mode**: Traditional operation using Cloud One API
  - Requires: `FSS_API_KEY` environment variable
  - Protocol: HTTP REST API
  - Endpoint: Cloud One File Security service
  
- **External Mode**: Connect to on-premise scanner
  - Requires: `SCANNER_EXTERNAL_ADDR` environment variable
  - Protocol: gRPC
  - No API key needed
  - Example: Kubernetes Vision One File Security

### 2. Dynamic Scanner Configuration
- Admin can change scanner settings via web UI
- Scanner process automatically restarts with new configuration
- No container restart required

### 3. TLS Support for External Scanners
- Optional TLS encryption for gRPC connections
- Configurable via `SCANNER_USE_TLS` environment variable or UI checkbox

## Usage

### Environment Variables

```bash
# External Scanner Mode
SCANNER_EXTERNAL_ADDR=10.10.21.201:50051
SCANNER_USE_TLS=false

# Cloud Mode (traditional)
FSS_API_KEY=your-api-key-here
FSS_API_ENDPOINT=antimalware.us-1.cloudone.trendmicro.com:443
```

### Web UI Configuration

1. Login as administrator
2. Navigate to Configuration page
3. Scroll to "Scanner Configuration" section
4. Enter external scanner address (e.g., `10.10.21.201:50051`)
5. Check "Use TLS" if required
6. Click "Save Configuration"
7. Scanner automatically restarts with new settings

## Testing

Run the provided test script:
```bash
./test-external-scanner.sh
```

This will:
1. Build the Docker image
2. Test cloud mode
3. Test external mode
4. Display logs showing which mode is active

## Files Modified

1. `scanner.go` - Go scanner service with dual-mode support
2. `server.js` - Backend API and configuration management
3. `public/configuration.html` - Admin UI with external scanner fields

## Files Added

1. `EXTERNAL_SCANNER.md` - User documentation
2. `test-external-scanner.sh` - Testing script
3. `IMPLEMENTATION.md` - This file (technical documentation)

## Backward Compatibility

✅ **Fully backward compatible**
- Existing deployments continue to work without changes
- Default mode is Cloud Scanner (existing behavior)
- External scanner is optional and only activated when configured

## Security Considerations

1. **No API Key Required**: External scanners don't need Cloud One API keys
2. **TLS Optional**: Can be enabled for encrypted gRPC connections
3. **Admin Only**: Scanner configuration requires administrator privileges
4. **Process Isolation**: Scanner runs as separate process with controlled restart

## Known Limitations

1. Scanner restart requires killing the existing process
2. Brief downtime (2 seconds) during scanner configuration changes
3. External scanner must be reachable from the container network

## Future Enhancements

- [ ] Health check for external scanner connectivity
- [ ] Automatic failover between cloud and external scanners
- [ ] Scanner connection pooling for high-performance scenarios
- [ ] Support for multiple external scanners (load balancing)
