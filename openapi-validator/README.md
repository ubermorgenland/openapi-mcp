# OpenAPI Validator Web Interface

A modern, responsive web interface for validating and linting OpenAPI specifications using the `openapi-mcp` HTTP API.

## Features

- **Drag & Drop File Upload**: Simply drag your OpenAPI file onto the upload area
- **Text Input**: Paste your OpenAPI specification directly into the text editor
- **Real-time Validation**: Validate against remote `openapi-mcp` servers
- **Comprehensive Linting**: Choose between basic validation or detailed linting
- **Detailed Results**: View errors and warnings with actionable suggestions
- **Export Results**: Download validation results as JSON
- **Responsive Design**: Works great on desktop and mobile devices

## Getting Started

### 1. Start the OpenAPI Validation Server

First, ensure you have the `openapi-mcp` tool running in HTTP mode:

```bash
# Start the HTTP server (both validate and lint endpoints will be available)
./openapi-mcp --http=:8080 validate
# OR
./openapi-mcp --http=:8080 lint

# Both commands now expose both /validate and /lint endpoints
# The web interface can switch between modes dynamically
```

### 2. Open the Web Interface

Simply open `index.html` in your web browser. You can:

- Open it directly from your file system
- Serve it with a local web server for better performance:

```bash
# Using Python
python -m http.server 8000

# Using Node.js (if you have http-server installed)
npx http-server

# Using PHP
php -S localhost:8000
```

### 3. Configure the Server URL

In the web interface:

1. Set the **Validation Server URL** to match your running server (default: `http://localhost:8080`)
2. Choose your **Validation Mode**:
   - **Validate**: Basic validation for critical issues
   - **Lint**: Comprehensive analysis with best practice suggestions
3. Click **Test Connection** to verify connectivity

### 4. Validate Your OpenAPI Spec

You can provide your OpenAPI specification in two ways:

#### Option A: File Upload
- Drag and drop your `.yaml`, `.yml`, or `.json` file onto the upload area
- Or click the upload area to browse for files

#### Option B: Text Input
- Paste your OpenAPI specification directly into the text editor
- Click **Load Example** to see a sample specification

Then click **Validate** or **Lint** to analyze your specification.

## Understanding Results

### Success
When validation passes, you'll see a green success message indicating your OpenAPI specification is valid.

### Issues
When issues are found, they're categorized as:

- **Errors** (🔴): Critical issues that prevent the specification from working
- **Warnings** (🟡): Best practice violations or potential improvements

Each issue includes:
- **Message**: Description of the problem
- **Suggestion**: Actionable advice for fixing the issue
- **Context**: Location information (operation, path, method, etc.)

### Export Results
Click **Export Results** to download the full validation report as a JSON file for further analysis or documentation.

## Server Configuration

The web interface can connect to any `openapi-mcp` server running with HTTP API enabled. Common configurations:

### Local Development
```bash
# Basic validation server
./openapi-mcp --http=:8080 validate

# Comprehensive linting server  
./openapi-mcp --http=:8080 lint
```

### Remote Server
Update the server URL in the web interface to point to your remote validation service:
```
https://your-domain.com:8080
```

### Custom Port
```bash
# Run on a different port
./openapi-mcp --http=:3000 validate
```

Then update the server URL to `http://localhost:3000`.

## Example OpenAPI Specifications

The interface includes a **Load Example** button that provides a sample Pet Store API specification. This is useful for:

- Testing the validation functionality
- Learning OpenAPI best practices
- Understanding the types of issues the linter can detect

## Browser Compatibility

This web interface works with all modern browsers that support:
- ES6+ JavaScript features
- CSS Grid and Flexbox
- Fetch API
- File API (for drag & drop)

Tested browsers:
- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## CORS Support

The `openapi-mcp` HTTP API includes full CORS support, allowing the web interface to connect from any origin. This means you can:

- Host the web interface on any domain
- Use it locally without CORS issues
- Deploy it to static hosting services like GitHub Pages, Netlify, etc.

## Deployment

To deploy this web interface:

1. **Static Hosting**: Upload the files to any static web hosting service
2. **GitHub Pages**: Push to a GitHub repository and enable Pages
3. **Netlify/Vercel**: Connect your repository for automatic deployments
4. **CDN**: Use any CDN service to serve the static files

No server-side configuration is needed - this is a pure client-side application.

## Troubleshooting

### Connection Issues
- Ensure the `openapi-mcp` server is running
- Check the server URL and port
- Verify firewall settings if using remote servers
- Use the **Test Connection** button to diagnose connectivity

### Validation Failures
- Check that your OpenAPI specification is valid YAML or JSON
- Ensure it follows OpenAPI 3.x format
- Review the detailed error messages for specific issues

### Browser Console
Open your browser's developer tools and check the console for any JavaScript errors or network issues.

## Contributing

To improve this web interface:

1. Modify the HTML structure in `index.html`
2. Update styling in `styles.css`
3. Enhance functionality in `script.js`
4. Test with various OpenAPI specifications
5. Ensure responsive design works across devices

The code is well-commented and follows modern web development practices.