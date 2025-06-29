#!/bin/bash

# Demo script for Internal Transfers API
set -e

API_BASE="http://localhost:8080"
echo "🚀 Internal Transfers API Demo"
echo "================================="

# Wait for service to be ready
echo "🔄 Waiting for service to be ready..."
for i in {1..30}; do
    if curl -s "$API_BASE/healthz" > /dev/null 2>&1; then
        echo "✅ Service is ready!"
        break
    fi
    echo "   Attempt $i/30..."
    sleep 2
done

# Test health endpoint
echo ""
echo "📝 Health Check"
curl -s "$API_BASE/healthz" | jq '.'

# Create accounts
echo ""
echo "🏦 Creating Accounts"
echo "==================="

echo "Creating Account 1 with $1000.00"
ACCOUNT1_RESPONSE=$(curl -s -X POST "$API_BASE/v1/accounts" \
    -H "Content-Type: application/json" \
    -d '{"initial_balance": "1000.00"}')
ACCOUNT1_ID=$(echo "$ACCOUNT1_RESPONSE" | jq -r '.id')
echo "Account 1 ID: $ACCOUNT1_ID"
echo "$ACCOUNT1_RESPONSE" | jq '.'

echo ""
echo "Creating Account 2 with $500.50"
ACCOUNT2_RESPONSE=$(curl -s -X POST "$API_BASE/v1/accounts" \
    -H "Content-Type: application/json" \
    -d '{"initial_balance": "500.50"}')
ACCOUNT2_ID=$(echo "$ACCOUNT2_RESPONSE" | jq -r '.id')
echo "Account 2 ID: $ACCOUNT2_ID"
echo "$ACCOUNT2_RESPONSE" | jq '.'

# Transfer money
echo ""
echo "💸 Money Transfer"
echo "=================="

echo "Transferring $150.25 from Account 1 to Account 2"
TRANSFER_RESPONSE=$(curl -s -X POST "$API_BASE/v1/transactions" \
    -H "Content-Type: application/json" \
    -d "{\"source_account_id\": \"$ACCOUNT1_ID\", \"destination_account_id\": \"$ACCOUNT2_ID\", \"amount\": \"150.25\", \"reference\": \"demo-payment\"}")
echo "$TRANSFER_RESPONSE" | jq '.'

# Check balances
echo ""
echo "📊 Updated Balances"
echo "==================="

echo "Account 1 balance:"
curl -s "$API_BASE/v1/accounts/$ACCOUNT1_ID" | jq '.'

echo ""
echo "Account 2 balance:"
curl -s "$API_BASE/v1/accounts/$ACCOUNT2_ID" | jq '.'

echo ""
echo "�� Demo completed!" 