# Test Connection Feature

## Overview
Added a "Test Connection" button to the external scanner configuration section, allowing administrators to verify connectivity to the external gRPC scanner before applying configuration changes.

## User Interface Changes

### Configuration Page
**Location**: Configuration → Scanner Configuration → External gRPC Scanner

**New Elements**:
- **Test Connection Button**: Blue button below the TLS checkbox
- **Result Display**: Inline status message showing connection test results

### Visual Feedback
- 🕒 **Testing**: Gray "⏳ Connecting..." message while testing
- ✓ **Success**: Green checkmark with success message
- ✗ **Failed**: Red X with detailed error message
- ⚠️ **Warning**: Yellow warning for invalid address format

## How It Works

### User Flow
1. Admin enters external scanner address (e.g., `10.10.21.201:50051`)
2. Optionally checks TLS if the scanner requires encryption
3. Clicks **Test Connection** button
4. Button shows "Testing..." state (disabled)
5. Result appears next to button within ~5 seconds
6. Admin can proceed to save configuration if test succeeds

### Backend Implementation

**API Endpoint**: `POST /api/test-scanner`

**Request Body**:
```json
{
  "externalScannerAddr": "10.10.21.201:50051",
  "externalScannerTLS": false
}
```

**Response (Success)**:
```json
{
  "success": true,
  "message": "Successfully connected to external scanner at 10.10.21.201:50051"
}
```

**Response (Failure)**:
```json
{
  "success": false,
  "message": "Cannot reach scanner at 10.10.21.201:50051. Check network connectivity."
}
```

### Test Process

1. **Validation**: Checks address format (host:port)
2. **Process Spawn**: Creates temporary scanner process with test environment
3. **Connection Attempt**: Scanner tries to initialize gRPC client
4. **Log Analysis**: Examines scanner output for success/error indicators
5. **Timeout**: 5-second timeout for connection attempt
6. **Cleanup**: Test process is terminated after check

### Detection Logic

The test examines scanner logs for:
- ✅ **Success Indicators**: "Scanner started in External Scanner mode"
- ❌ **Failure Indicators**: "connection refused", "no such host", "timeout"
- ⚠️ **Format Errors**: Invalid host:port format

## Benefits

1. **Pre-validation**: Verify connectivity before saving configuration
2. **Troubleshooting**: Get immediate feedback on connection issues
3. **Network Testing**: Confirm firewall rules and routing
4. **Time Saving**: Avoid container restarts for misconfigured settings
5. **User Confidence**: Clear visual feedback on scanner accessibility

## Error Messages

| Error | Meaning | Solution |
|-------|---------|----------|
| "Connection timeout after 5 seconds" | Scanner not responding | Check if scanner is running, verify network route |
| "Cannot reach scanner at..." | Network connectivity issue | Verify IP address, check firewall rules |
| "Invalid address format" | Malformed host:port | Use format: `10.10.21.201:50051` |
| "Connection error: connection refused" | Scanner not listening on port | Verify port number (50051 for gRPC) |
| "External scanner address is required" | Empty address field | Enter scanner address before testing |

## Technical Details

### Frontend Code (configuration.html)
```javascript
function testScannerConnection() {
    const externalScannerAddr = document.getElementById('externalScannerAddr').value.trim();
    const externalScannerTLS = document.getElementById('externalScannerTLS').checked;
    
    fetch('/api/test-scanner', {
        method: 'POST',
        headers: {
            'Authorization': 'Basic ' + btoa(username + ':' + password),
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ externalScannerAddr, externalScannerTLS })
    })
    .then(response => response.json())
    .then(data => {
        // Display result with appropriate styling
    });
}
```

### Backend Code (server.js)
```javascript
app.post('/api/test-scanner', combinedAuth, adminAuth, async (req, res) => {
    // Validate address format
    // Spawn temporary scanner process
    // Monitor process output
    // Return success/failure result
});
```

## Usage Example

### Scenario 1: Testing Kubernetes Scanner
```
1. User enters: 10.10.21.201:50051
2. User unchecks TLS (not required)
3. User clicks "Test Connection"
4. System shows: ⏳ Connecting...
5. After 2 seconds: ✓ Successfully connected to external scanner at 10.10.21.201:50051
6. User clicks "Save Configuration"
```

### Scenario 2: Wrong Port Number
```
1. User enters: 10.10.21.201:3001 (HTTP port instead of gRPC)
2. User clicks "Test Connection"
3. System shows: ⏳ Connecting...
4. After 5 seconds: ✗ Connection timeout after 5 seconds
5. User corrects to: 10.10.21.201:50051
6. Test succeeds
```

### Scenario 3: Network Unreachable
```
1. User enters: 192.168.99.99:50051 (incorrect IP)
2. User clicks "Test Connection"
3. System shows: ⏳ Connecting...
4. After 3 seconds: ✗ Cannot reach scanner at 192.168.99.99:50051. Check network connectivity.
5. User verifies IP address with IT team
```

## Security Considerations

1. **Admin-Only**: Test endpoint requires administrator authentication
2. **Process Isolation**: Test runs in temporary process, doesn't affect main scanner
3. **Timeout Protection**: 5-second timeout prevents hanging connections
4. **No Sensitive Data**: Test doesn't expose API keys or credentials
5. **Log Privacy**: Error messages don't reveal system internals

## Future Enhancements

- [ ] Show more detailed connection metrics (latency, version)
- [ ] Support testing cloud scanner API key validity
- [ ] Add visual indicator for scanner health status
- [ ] Cache last successful test result
- [ ] Add retry mechanism for transient failures
- [ ] Display scanner version information if available

## Files Modified

1. **server.js** (Lines ~441-537)
   - Added `POST /api/test-scanner` endpoint
   - Address validation with regex
   - Temporary process spawning for connection test
   - 5-second timeout with graceful failure

2. **configuration.html** (Lines ~302-309)
   - Added "Test Connection" button
   - Added result display span
   - Styled button with blue theme

3. **configuration.html** (Lines ~547-588)
   - Added `testScannerConnection()` JavaScript function
   - Fetch API call to `/api/test-scanner`
   - Dynamic result display with color-coded messages
   - Button state management (disabled during test)

4. **EXTERNAL_SCANNER.md**
   - Added "Testing Scanner Connection" section
   - Usage instructions with visual indicators
   - Troubleshooting tips

5. **README.md**
   - Updated External Scanner Mode features list
   - Mentioned built-in connection test

## Testing Checklist

- [x] Test with valid external scanner address
- [x] Test with invalid IP address
- [x] Test with wrong port number
- [x] Test with invalid format (missing port)
- [x] Test with empty address field
- [x] Test TLS enabled vs disabled
- [x] Verify button disabled state during test
- [x] Verify timeout after 5 seconds
- [x] Verify error messages are user-friendly
- [x] Verify admin-only access restriction
