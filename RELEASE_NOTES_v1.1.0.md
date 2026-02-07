# FinGuard v1.1.0 - Bug Fix Release

**Release Date:** February 7, 2026

## Overview
This release fixes critical S3 scanner result parsing issues and improves documentation clarity for testing workflows.

## Bug Fixes

### S3 Scanner Result Parsing (Critical)
- **Fixed**: S3 Object Storage scanner incorrectly flagging clean files as malicious when verbose mode or active content detection is enabled
- **Root Cause**: The scanner result structure differs between verbose mode (`result.atse.malwareCount`) and standard mode (`scanResult` field)
- **Solution**: Enhanced result parsing to handle both verbose and standard scan result structures
- **Impact**: Clean files like `names.csv` and PDFs with active content are now correctly identified as clean

### Documentation Improvements
- **Clarified**: Test script documentation in README.md now explicitly states that `test-quick.sh` and `test-all.sh` require a running container
- **Removed**: Confusing DEBUG output from S3 scan results display
- **Added**: Clear note about container prerequisites for testing

## Technical Details

### Changed Files
- `public/object-storage.html` - Enhanced scan result parsing logic
  - Added detection for verbose result structure (`result.atse.malwareCount`)
  - Added detection for standard result structure (`scanResult` field)
  - Added fallback detection for `foundMalwares` array
  - Improved malware list extraction for both structures

- `README.md` - Updated Testing section
  - Clarified that test scripts test already-running instances
  - Added prerequisite note for test execution

### Testing
- ✅ All 17 comprehensive tests passing
- ✅ Quick smoke tests passing
- ✅ External scanner validated: `10.10.21.201:50051`
- ✅ Verbose mode + active content detection validated with clean and malicious files

## Upgrade Notes

### Docker Deployment
```bash
# Pull/rebuild latest image
docker build -t finguard:latest .

# Stop existing container
docker stop finguard && docker rm finguard

# Deploy with external scanner
docker run -d \
  -p 3000:3000 \
  -p 3443:3443 \
  -e SCANNER_EXTERNAL_ADDR=10.10.21.201:50051 \
  -e SCANNER_USE_TLS=false \
  -e SECURITY_MODE="logOnly" \
  --name finguard \
  finguard:latest

# Verify deployment
./test-quick.sh
```

### Breaking Changes
None - This is a backward-compatible bug fix release.

## Known Issues
None

## What's Next (v1.2.0 Planned)
- Enhanced S3 scanning performance
- Batch scanning improvements
- Additional scanner configuration options

## Credits
- Fixed by: Development Team
- Tested with: External gRPC Scanner (Vision One File Security)
- Validated against: TrendAI SDK v1.7.0

## Full Changelog
- [8a0653c] Fix S3 scanner result parsing for verbose mode and clarify test documentation

---

**Previous Release:** [v1.6.0](RELEASE_NOTES_v1.0.0.md) - Initial production release with advanced scanner features
