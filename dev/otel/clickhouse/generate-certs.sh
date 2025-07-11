#!/bin/sh
set -e

# Configuration
CERT_DIR="${CERT_DIR:-/certs}"
CERT_FILE="$CERT_DIR/server.crt"
KEY_FILE="$CERT_DIR/server.key"
DH_FILE="$CERT_DIR/dhparam.pem"

echo "=== ClickHouse SSL Certificate Generator ==="
echo "Certificate directory: $CERT_DIR"

# Create the certificate directory if it doesn't exist
mkdir -p "$CERT_DIR"

# Check if certificates already exist and are valid
if [ -f "$CERT_FILE" ] && [ -f "$KEY_FILE" ] && [ -f "$DH_FILE" ]; then
    echo "Found existing SSL certificates, checking validity..."
    
    # Check if certificate is still valid (expires in more than 24 hours)
    if openssl x509 -checkend 86400 -noout -in "$CERT_FILE" >/dev/null 2>&1; then
        echo "✓ Existing certificates are valid (expires in >24h), skipping generation."
        echo "Certificate details:"
        openssl x509 -in "$CERT_FILE" -text -noout | grep -E "(Subject:|Not Before:|Not After:)" || true
        exit 0
    else
        echo "⚠ Existing certificates are expired or will expire soon, regenerating..."
        # Remove old certificates to ensure clean generation
        rm -f "$CERT_FILE" "$KEY_FILE" "$DH_FILE"
    fi
else
    echo "No existing certificates found, generating new ones..."
fi

# Ensure we have OpenSSL
if ! command -v openssl 2>/dev/null; then
    echo "Error: OpenSSL is not installed or not in PATH"
    exit 1
fi

# Generate private key
echo "🔑 Generating RSA private key..."
if ! openssl genrsa -out "$KEY_FILE" 2048; then
    echo "Error: Failed to generate private key"
    exit 1
fi

# Generate self-signed certificate valid for 365 days
# Include multiple subject alternative names for flexibility
echo "📜 Generating self-signed certificate..."
if ! openssl req -new -x509 -key "$KEY_FILE" -out "$CERT_FILE" -days 365 \
    -subj "/CN=clickhouse-server/O=PMM Dev Environment/C=US" \
    -addext "subjectAltName=DNS:clickhouse-server,DNS:pmm-server,DNS:localhost,DNS:otel-collector,IP:127.0.0.1,IP:::1"; then
    echo "Error: Failed to generate certificate"
    exit 1
fi

# Generate DH parameters (this can take a while)
echo "🔐 Generating DH parameters (this may take a few minutes)..."
if ! openssl dhparam -out "$DH_FILE" 2048; then
    echo "Error: Failed to generate DH parameters"
    exit 1
fi

# Set appropriate permissions for security
echo "🔒 Setting file permissions..."
chmod 600 "$KEY_FILE"    # Private key should be readable only by owner
chmod 644 "$CERT_FILE"   # Certificate can be readable by all
chmod 644 "$DH_FILE"     # DH params can be readable by all

# Verify file ownership and permissions
if [ -n "${CHOWN_TO:-}" ]; then
    echo "📁 Setting ownership to: $CHOWN_TO"
    chown "$CHOWN_TO" "$CERT_FILE" "$KEY_FILE" "$DH_FILE"
fi

echo ""
echo "✅ SSL certificates generated successfully!"
echo "📁 Files created:"
echo "   Certificate: $CERT_FILE"
echo "   Private Key: $KEY_FILE"
echo "   DH Params:   $DH_FILE"
echo ""

# Display certificate information
echo "📋 Certificate details:"
echo "----------------------------------------"
if openssl x509 -in "$CERT_FILE" -text -noout | grep -E "(Subject:|Issuer:|Not Before:|Not After:|DNS:|IP Address:)" 2>/dev/null; then
    :  # Success
else
    echo "Certificate appears to be valid but details extraction failed"
fi
echo "----------------------------------------"
echo ""
echo "🎉 Certificate generation completed successfully!"
