#!/bin/bash

# Test database connection and API key retrieval
export DATABASE_URL="postgresql://apinferenceadmin:%402rv68%3AB%24LxSvch%7C@127.0.0.1:5432/database-apinference-app?sslmode=disable"

echo "Testing database connection and weather API key..."

# Test 1: Can we connect to database?
echo "1. Testing database connection:"
if psql "$DATABASE_URL" -c "SELECT 1;" > /dev/null 2>&1; then
    echo "✅ Database connection successful"
else
    echo "❌ Database connection failed"
    echo "This explains why database fallback authentication fails"
    exit 1
fi

echo ""

# Test 2: Does weather spec exist?
echo "2. Checking for weather spec:"
weather_count=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM openapi_specs WHERE spec_name = 'weather';" 2>/dev/null | xargs)
if [ "$weather_count" -gt 0 ]; then
    echo "✅ Weather spec found in database"
else
    echo "❌ Weather spec not found in database"
    echo "Need to insert weather spec first"
fi

echo ""

# Test 3: Does weather spec have API key?
echo "3. Checking weather API key:"
api_key=$(psql "$DATABASE_URL" -t -c "SELECT api_key_token FROM openapi_specs WHERE spec_name = 'weather';" 2>/dev/null | xargs)
if [ -n "$api_key" ] && [ "$api_key" != "" ]; then
    # Mask the key for security
    masked_key="${api_key:0:4}****${api_key: -4}"
    echo "✅ Weather API key found: $masked_key"
else
    echo "❌ Weather API key not found or empty"
    echo "Setting weather API key to: YOUR_WEATHER_API_KEY"
    
    # Insert or update weather API key
    psql "$DATABASE_URL" -c "
        INSERT INTO openapi_specs (spec_name, api_key_token, is_active, created_at, updated_at) 
        VALUES ('weather', 'YOUR_WEATHER_API_KEY_PLACEHOLDER', true, NOW(), NOW())
        ON CONFLICT (spec_name) 
        DO UPDATE SET 
            api_key_token = EXCLUDED.api_key_token,
            updated_at = NOW();
    " > /dev/null 2>&1
    
    # Verify it was set
    new_key=$(psql "$DATABASE_URL" -t -c "SELECT api_key_token FROM openapi_specs WHERE spec_name = 'weather';" 2>/dev/null | xargs)
    if [ "$new_key" = "YOUR_WEATHER_API_KEY_PLACEHOLDER" ]; then
        echo "✅ Weather API key successfully set in database"
    else
        echo "❌ Failed to set weather API key in database"
    fi
fi

echo ""
echo "4. Current database state:"
psql "$DATABASE_URL" -c "SELECT spec_name, 
    CASE WHEN api_key_token IS NOT NULL AND LENGTH(api_key_token) > 8 
         THEN SUBSTRING(api_key_token FROM 1 FOR 4) || '****' || SUBSTRING(api_key_token FROM LENGTH(api_key_token)-3) 
         ELSE 'NO_KEY' 
    END as masked_key,
    is_active,
    updated_at 
FROM openapi_specs 
ORDER BY spec_name;" 2>/dev/null || echo "Failed to query database"

echo ""
echo "💡 If database connection works and API key is set, then database fallback authentication should work"
echo "💡 If tests still fail, the issue is likely in the authentication context creation or HTTP client usage"