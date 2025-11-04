# OpenAPI MCP Authentication Test Suite

This directory contains comprehensive authentication tests for the OpenAPI MCP server.

## Centralized Secrets Management

All test scripts now use a centralized secrets system:

### Files:
- `.env.test` - Contains all API keys and tokens (⚠️ NOT committed to git)
- `load_test_secrets.sh` - Utility script to load and validate secrets
- `logs/` - Directory for test logs and reports

### Setup:
1. Update `.env.test` with your actual API credentials
2. All test scripts automatically source `load_test_secrets.sh`
3. Logs are written to the `logs/` directory

### API Keys Required:
- `TWITTER_RAPIDAPI_KEY` - Twitter API via RapidAPI
- `GOOGLE_ANALYTICS_BEARER_TOKEN` - Google Analytics OAuth token
- `GOOGLE_FINANCE_RAPIDAPI_KEY` - Google Finance via RapidAPI
- `GOOGLE_KEYWORDS_RAPIDAPI_KEY` - Google Keywords via RapidAPI
- `ALPHA_VANTAGE_API_KEY` - Alpha Vantage API key
- `WEATHER_API_KEY` - Weather API key
- `PERPLEXITY_BEARER_TOKEN` - Perplexity AI Bearer token
- `YOUTUBE_RAPIDAPI_KEY` - YouTube Transcript via RapidAPI

### Running Tests:
```bash
# Test individual APIs
./test_weather_streamable.sh
./test_google_analytics_sse.sh

# Test all APIs
./test_all_specs.sh

# Test concurrent authentication
./test_concurrent_auth.sh
```

### Authentication Priority:
1. **Tool Arguments** (highest priority)
2. **HTTP Headers** 
3. **Database Fallback** (lowest priority)

All tests verify this authentication hierarchy works correctly.