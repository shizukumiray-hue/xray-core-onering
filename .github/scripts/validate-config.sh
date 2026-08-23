#!/bin/bash
# Validate Multi-CDN configuration file

set -e

CONFIG_FILE="${1:-config.json}"

if [[ ! -f "$CONFIG_FILE" ]]; then
    echo "❌ Error: Config file not found: $CONFIG_FILE"
    exit 1
fi

echo "🔍 Validating Multi-CDN Configuration: $CONFIG_FILE"
echo ""

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "⚠️  Warning: jq not installed. Skipping JSON validation."
    echo "   Install: sudo apt install jq (Linux) or brew install jq (macOS)"
else
    # Validate JSON syntax
    if jq empty "$CONFIG_FILE" 2>/dev/null; then
        echo "✅ JSON syntax valid"
    else
        echo "❌ JSON syntax error"
        exit 1
    fi
    
    # Check Multi-CDN configuration
    if jq -e '.outbounds[0].streamSettings.tlsSettings.multiCDN' "$CONFIG_FILE" &> /dev/null; then
        echo "✅ Multi-CDN configuration found"
        
        # Check if enabled
        ENABLED=$(jq -r '.outbounds[0].streamSettings.tlsSettings.multiCDN.enabled' "$CONFIG_FILE")
        if [[ "$ENABLED" == "true" ]]; then
            echo "✅ Multi-CDN enabled"
            
            # Check providers
            PROVIDER_COUNT=$(jq -r '.outbounds[0].streamSettings.tlsSettings.multiCDN.providers | length' "$CONFIG_FILE")
            echo "✅ CDN providers: $PROVIDER_COUNT"
            
            if [[ "$PROVIDER_COUNT" -lt 2 ]]; then
                echo "⚠️  Warning: Less than 2 CDN providers configured (recommended: 2+)"
            fi
            
            # Check strategy
            STRATEGY=$(jq -r '.outbounds[0].streamSettings.tlsSettings.multiCDN.strategy // "round-robin"' "$CONFIG_FILE")
            echo "✅ Selection strategy: $STRATEGY"
            
            # Check health check
            HEALTH_CHECK=$(jq -r '.outbounds[0].streamSettings.tlsSettings.multiCDN.healthCheck.enabled // false' "$CONFIG_FILE")
            if [[ "$HEALTH_CHECK" == "true" ]]; then
                echo "✅ Health check enabled"
            else
                echo "⚠️  Warning: Health check disabled (recommended: enabled)"
            fi
            
            # Check evasion
            EVASION=$(jq -r '.outbounds[0].streamSettings.tlsSettings.multiCDN.evasion.enableRotation // false' "$CONFIG_FILE")
            if [[ "$EVASION" == "true" ]]; then
                echo "✅ CDN rotation enabled"
            else
                echo "ℹ️  Info: CDN rotation disabled"
            fi
            
        else
            echo "⚠️  Multi-CDN found but disabled"
        fi
    else
        echo "ℹ️  No Multi-CDN configuration (using standard mode)"
    fi
    
    # Check serverName format
    SERVER_NAME=$(jq -r '.outbounds[0].streamSettings.tlsSettings.serverName // ""' "$CONFIG_FILE")
    if [[ "$SERVER_NAME" =~ ^onering-multi: ]]; then
        echo "✅ Multi-CDN serverName format detected: $SERVER_NAME"
    elif [[ "$SERVER_NAME" =~ ^onering: ]]; then
        echo "ℹ️  Single-CDN serverName format: $SERVER_NAME"
    else
        echo "ℹ️  Standard serverName: $SERVER_NAME"
    fi
fi

echo ""
echo "🎉 Validation complete!"
