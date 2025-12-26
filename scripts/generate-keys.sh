#!/bin/bash

if ! command -v openssl >/dev/null 2>&1; then
    echo "Error: openssl is not installed. Please install it to generate keys."
    exit 1
fi

echo ">> Generating VAPID Keys..."

# 1. Generate a P-256 private key
openssl ecparam -name prime256v1 -genkey -noout -out private.pem

# 2. Extract the Public Key, convert to uncompressed DER format, and Base64 URL-safe encode
PUB_KEY=$(openssl ec -in private.pem -pubout -outform DER 2>/dev/null | tail -c 65 | base64 | tr '+/' '-_' | tr -d '=')

# 3. Extract the Private Key (32-byte raw) and Base64 URL-safe encode
PRIV_KEY=$(openssl ec -in private.pem -outform DER 2>/dev/null | tail -c +8 | head -c 32 | base64 | tr '+/' '-_' | tr -d '=')

# 4. Clean up the temporary file
rm private.pem

echo ""
echo "Copy these into your .env file:"
echo "----------------------------------------"
echo "VAPID_PUBLIC_KEY=$PUB_KEY"
echo "VAPID_PRIVATE_KEY=$PRIV_KEY"
echo "----------------------------------------"
