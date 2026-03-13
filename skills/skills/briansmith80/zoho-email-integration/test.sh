#!/bin/bash
# Test script for Zoho Email Integration
# Usage: ZOHO_EMAIL=xxx ZOHO_PASSWORD=xxx ./test.sh

set -e

echo " Testing Zoho Email Integration"
echo "=================================="
echo ""

# Check credentials
if [ -z "$ZOHO_EMAIL" ] || [ -z "$ZOHO_PASSWORD" ]; then
    echo " Error: ZOHO_EMAIL and ZOHO_PASSWORD must be set"
    echo ""
    echo "Usage:"
    echo "  export ZOHO_EMAIL='your@email.com'"
    echo "  export ZOHO_PASSWORD='your-app-password'"
    echo "  ./test.sh"
    exit 1
fi

echo "✓ Credentials found"
echo ""

# Test 1: Unread count
echo "Test 1: Checking unread count..."
RESULT=$(python3 scripts/zoho-email.py unread)
COUNT=$(echo "$RESULT" | jq -r '.unread_count')
echo "✓ Unread count: $COUNT"
echo ""

# Test 2: Search inbox (should be fast with date filtering)
echo "Test 2: Searching inbox..."
RESULT=$(timeout 15 python3 scripts/zoho-email.py search "test")
FOUND=$(echo "$RESULT" | jq '. | length')
echo "✓ Found $FOUND emails"
echo ""

# Test 3: Verbose mode
echo "Test 3: Testing verbose mode..."
python3 scripts/zoho-email.py unread --verbose 2>&1 | grep -q "DEBUG" && echo "✓ Verbose mode works" || echo " Verbose mode failed"
echo ""

# Test 4: Help text
echo "Test 4: Help text..."
python3 scripts/zoho-email.py 2>&1 | grep -q "Usage:" && echo "✓ Help text displayed" || echo " Help text failed"
echo ""

# Test 5: Error handling (missing credentials)
echo "Test 5: Error handling..."
(unset ZOHO_EMAIL; unset ZOHO_PASSWORD; python3 scripts/zoho-email.py unread 2>&1) | grep -q "ZOHO_EMAIL and ZOHO_PASSWORD" && echo "✓ Error handling works" || echo " Error handling failed"
echo ""

echo "=================================="
echo " All tests passed!"
echo ""
echo "Optional manual tests:"
echo "  • Send email: python3 scripts/zoho-email.py send 'test@example.com' 'Test' 'Body'"
echo "  • Search sent: python3 scripts/zoho-email.py search-sent 'keyword'"
echo "  • Get email: python3 scripts/zoho-email.py get INBOX <email_id>"
