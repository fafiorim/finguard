#!/bin/bash

# View S3 Scanner Logs
# This script shows the dedicated S3 scanning logs

echo "=== S3 Scanner Logs ==="
echo ""
echo "Options:"
echo "  1) View all S3 logs"
echo "  2) Follow S3 logs (live)"
echo "  3) Search logs"
echo ""

case "$1" in
    "")
        # Default: show all logs
        echo "Showing all S3 logs:"
        echo "===================="
        docker exec finguard-external cat /var/log/s3-scanner.log 2>/dev/null || echo "Log file not found or container not running"
        ;;
    "follow"|"-f")
        # Follow logs in real-time
        echo "Following S3 logs (Ctrl+C to stop):"
        echo "===================================="
        docker exec finguard-external tail -f /var/log/s3-scanner.log
        ;;
    "search")
        # Search logs
        if [ -z "$2" ]; then
            echo "Usage: $0 search <pattern>"
            exit 1
        fi
        echo "Searching for: $2"
        echo "===================="
        docker exec finguard-external grep "$2" /var/log/s3-scanner.log
        ;;
    "last")
        # Show last N lines
        LINES=${2:-50}
        echo "Showing last $LINES lines:"
        echo "=========================="
        docker exec finguard-external tail -n $LINES /var/log/s3-scanner.log
        ;;
    "errors")
        # Show only errors
        echo "Showing only ERROR messages:"
        echo "============================"
        docker exec finguard-external grep "ERROR" /var/log/s3-scanner.log
        ;;
    "scans")
        # Show only scan operations
        echo "Showing scan operations:"
        echo "========================"
        docker exec finguard-external grep "SCAN REQUEST\|Scan completed\|Scan result:" /var/log/s3-scanner.log
        ;;
    *)
        echo "Usage: $0 [follow|search <pattern>|last [N]|errors|scans]"
        echo ""
        echo "Examples:"
        echo "  $0              # View all logs"
        echo "  $0 follow       # Follow logs in real-time"
        echo "  $0 search 2600  # Search for '2600' in logs"
        echo "  $0 last 100     # Show last 100 lines"
        echo "  $0 errors       # Show only errors"
        echo "  $0 scans        # Show only scan operations"
        exit 1
        ;;
esac
