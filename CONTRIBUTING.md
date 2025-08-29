# Contributing to openapi-mcp

Thank you for your interest in contributing to openapi-mcp! We welcome contributions from the community and appreciate your help in making this project better.

## 🤝 Code of Conduct

We are committed to providing a welcoming and inclusive environment for all contributors. Please be respectful, constructive, and collaborative in all interactions.

**Our Community Values:**
- **Respect**: Treat all contributors with respect, regardless of experience level
- **Collaboration**: Work together to solve problems and improve the project
- **Learning**: Help newcomers and share knowledge openly
- **Quality**: Maintain high standards while being patient with learning
- **Inclusivity**: Welcome diverse perspectives and backgrounds

## 🚀 Getting Started

### Prerequisites

- **Go 1.22+**: Ensure you have a recent Go version installed
- **Git**: For version control and submitting changes
- **PostgreSQL** (optional): For database-driven features testing

### Development Setup

1. **Fork and Clone**:
   ```sh
   git clone https://github.com/yourusername/openapi-mcp.git
   cd openapi-mcp
   ```

2. **Install Dependencies**:
   ```sh
   go mod download
   ```

3. **Build the Project**:
   ```sh
   make all
   # Creates binaries in bin/: openapi-mcp, mcp-client, spec-manager, etc.
   ```

4. **Run Tests**:
   ```sh
   go test ./...
   ```

5. **Set Up Database** (optional, for database features):
   ```sh
   # Start PostgreSQL and create a database
   export DATABASE_URL="postgresql://username:password@localhost:5432/openapi_mcp_dev"
   make seed-database
   ```

## 📝 Types of Contributions

We welcome various types of contributions:

### 🐛 Bug Reports
- **Use GitHub Issues** with the bug template
- **Include**: Go version, OS, reproduction steps, expected vs actual behavior
- **Provide**: Minimal example that reproduces the issue
- **Check**: Search existing issues first to avoid duplicates

### ✨ Feature Requests
- **Use GitHub Issues** with the feature template  
- **Describe**: The problem you're solving and proposed solution
- **Consider**: Breaking changes, backwards compatibility, maintenance burden
- **Discuss**: Large features in issues before implementing

### 🔧 Code Contributions
- **Start Small**: Fix bugs, improve documentation, add tests
- **Follow Conventions**: Match existing code style and patterns
- **Write Tests**: All new code should include comprehensive tests
- **Update Docs**: Keep documentation in sync with changes

### 📚 Documentation Improvements
- **README**: Keep usage examples current and clear
- **Godoc Comments**: Improve function and package documentation
- **Guides**: Add tutorials, best practices, troubleshooting guides
- **Examples**: Create real-world usage examples

## 🛠️ Development Workflow

### 1. **Create a Branch**
```sh
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-description
```

### 2. **Make Your Changes**
- Follow existing code patterns and naming conventions
- Write clear, self-documenting code
- Add comprehensive tests for new functionality
- Update documentation as needed

### 3. **Test Your Changes**
```sh
# Run all tests
go test ./...

# Run linting
go fmt ./...
go vet ./...

# Test with real OpenAPI specs
bin/openapi-mcp examples/weather.yaml

# Test database features (if applicable)
make test-with-database
```

### 4. **Commit Your Changes**
```sh
git add .
git commit -m "feat: add support for custom authentication headers

- Add X-Custom-Auth header support in HTTP mode
- Update documentation with examples
- Add tests for new authentication method

Fixes #123"
```

**Commit Message Format:**
- Use conventional commits: `type: description`
- Types: `feat`, `fix`, `docs`, `test`, `refactor`, `style`, `chore`
- Keep first line under 72 characters
- Add detailed description and reference issues when applicable

### 5. **Submit a Pull Request**
- Push your branch: `git push origin feature/your-feature-name`
- Open a PR with a clear title and description
- Link related issues using "Fixes #123" or "Closes #456"
- Respond to review feedback promptly and constructively

## 📋 Pull Request Guidelines

### Before Submitting
- [ ] All tests pass locally (`go test ./...`)
- [ ] Code is properly formatted (`go fmt ./...`)
- [ ] No linting errors (`go vet ./...`)
- [ ] Documentation is updated for new features
- [ ] Commit messages follow conventional format
- [ ] Changes are focused and atomic (one feature/fix per PR)

### PR Description Template
```markdown
## What This PR Does
Brief description of the changes and why they're needed.

## Changes Made
- [ ] Added/Modified: List specific changes
- [ ] Tests: Describe test coverage
- [ ] Docs: Note documentation updates

## Testing
Describe how you tested the changes:
- [ ] Unit tests pass
- [ ] Integration tests pass (if applicable)
- [ ] Manual testing with real OpenAPI specs

## Breaking Changes
List any breaking changes and migration steps (if any).

Fixes #issue_number
```

### Review Process
1. **Automated Checks**: CI must pass (build, tests, linting)
2. **Code Review**: At least one maintainer review required
3. **Testing**: Reviewers may test functionality manually
4. **Documentation**: Ensure docs are clear and complete
5. **Merge**: Squash-merge preferred for clean history

## 🏗️ Architecture Guidelines

### Code Organization
- **`cmd/`**: Executable entry points (main packages)
- **`pkg/`**: Reusable library packages
- **`internal/`**: Private implementation details
- **`examples/`**: Example OpenAPI specifications
- **`docs/`**: Additional documentation and guides

### Package Design Principles
- **Single Responsibility**: Each package has one clear purpose
- **Minimal Dependencies**: Avoid unnecessary external dependencies
- **Interface-Based Design**: Use interfaces for testability and flexibility
- **Error Handling**: Return meaningful errors with context
- **Documentation**: All exported items must have godoc comments

### Code Style Guidelines
- **Follow Go Conventions**: Use `go fmt`, `go vet`, and common Go idioms
- **Naming**: Use clear, descriptive names (avoid abbreviations)
- **Functions**: Keep functions small and focused (< 50 lines when possible)
- **Comments**: Write godoc for all exported items, inline comments for complex logic
- **Testing**: Aim for good test coverage, especially for public APIs

### Specific Areas for Contribution

#### 🔄 **Core Features**
- OpenAPI 3.1 support enhancements
- New authentication schemes
- Performance optimizations
- Error handling improvements

#### 🌐 **Transport Layer**
- WebSocket transport support
- gRPC transport implementation
- Custom protocol adapters
- Connection pooling and management

#### 🗄️ **Database Integration**
- New database backends (MySQL, SQLite, MongoDB)
- Migration system improvements
- Performance optimizations
- Connection management enhancements

#### 🧪 **Testing & Quality**
- Integration test suite expansion
- Performance benchmarking
- Fuzzing tests for OpenAPI parsing
- Mock server implementations

#### 📖 **Documentation**
- Tutorial and getting-started guides
- Real-world integration examples
- Video tutorials and demos
- API reference improvements

#### 🔌 **Integrations**
- IDE plugins and extensions
- CI/CD pipeline examples
- Docker and Kubernetes configs
- Cloud deployment guides

## 🐞 Debugging and Testing

### Running Tests Locally
```sh
# All tests
go test ./...

# Specific package
go test ./pkg/openapi2mcp

# Verbose output
go test -v ./...

# With race detection
go test -race ./...

# With coverage
go test -cover ./...
```

### Testing with Real APIs
```sh
# Test with weather API
export WEATHER_API_KEY="your_key"
bin/openapi-mcp examples/weather.yaml

# Test database features
export DATABASE_URL="postgresql://user:pass@localhost/test_db"
make seed-database
bin/openapi-mcp --http :8080
```

### Common Issues and Solutions

**Build Errors:**
- Run `go mod tidy` to clean dependencies
- Check Go version compatibility (1.22+ required)
- Verify all imports are accessible

**Test Failures:**
- Ensure PostgreSQL is running (for database tests)
- Check environment variables are set correctly
- Run tests in isolation: `go test -count=1 ./pkg/package`

**Import Path Issues:**
- Use `github.com/ubermorgenland/openapi-mcp/pkg/...` for imports
- Don't use relative imports within the project

## 💬 Getting Help

### Communication Channels
- **GitHub Issues**: For bug reports, feature requests, and technical questions
- **GitHub Discussions**: For general questions, ideas, and community chat
- **Pull Request Comments**: For code review discussions

### Asking for Help
When asking questions, please provide:
- **Context**: What you're trying to achieve
- **Environment**: Go version, OS, relevant configuration
- **Attempts**: What you've already tried
- **Code**: Minimal reproducible examples when applicable

### Mentorship for New Contributors
We're happy to mentor new contributors! Look for issues labeled:
- `good-first-issue`: Perfect for newcomers
- `help-wanted`: We need community help
- `documentation`: Great for learning the codebase
- `beginner-friendly`: Low complexity, well-defined scope

## 🏆 Recognition

We value all contributions and will recognize them through:
- **Contributors List**: All contributors listed in README
- **Release Notes**: Significant contributions highlighted
- **GitHub Recognition**: Appropriate reactions and comments
- **Community Mentions**: Sharing great contributions with the community

## 📚 Additional Resources

### Learning Resources
- **Go Documentation**: https://golang.org/doc/
- **MCP Protocol**: https://modelcontextprotocol.io/
- **OpenAPI Specification**: https://spec.openapis.org/oas/v3.0.3/
- **JSON-RPC 2.0**: https://www.jsonrpc.org/specification

### Project-Specific Guides
- **Database Setup**: See [DATABASE_SETUP.md](DATABASE_SETUP.md)
- **API Examples**: See [SPEC_API_EXAMPLES.md](SPEC_API_EXAMPLES.md)
- **Architecture**: See [docs/architecture.md](docs/architecture.md) (if available)

---

**Thank you for contributing to openapi-mcp!** 🎉

Your contributions help make AI-API integration more accessible and powerful for developers worldwide.