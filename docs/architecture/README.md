# OpenAPI MCP Architecture Documentation

This directory contains comprehensive architecture documentation for the OpenAPI MCP system, including security improvements, memory optimizations, and modular design patterns implemented to enhance scalability and maintainability.

## Documentation Index

### Core Architecture
- [**System Overview**](./system-overview.md) - High-level system architecture and component interactions
- [**Module Dependencies**](./module-dependencies.md) - Package structure and dependency relationships
- [**Data Flow**](./data-flow.md) - Request processing and data transformation flows

### Security Architecture  
- [**Authentication System**](./authentication.md) - Secure, context-based authentication without global state mutation
- [**Security Improvements**](./security-improvements.md) - Critical security fixes and threat mitigation

### Performance & Memory
- [**Memory Optimization**](./memory-optimization.md) - Memory-efficient processing for large API specifications
- [**Performance Patterns**](./performance-patterns.md) - Optimization strategies and resource management

### Error Handling
- [**Error Architecture**](./error-handling.md) - Structured error handling with context and tracing

## Quick Start

Each documentation file includes:
- **Mermaid diagrams** for visual representation
- **Code examples** demonstrating key patterns
- **Best practices** and implementation guidelines
- **Migration notes** from the previous architecture

## Architecture Principles

The redesigned architecture follows these core principles:

1. **Security First**: Eliminate global state mutations and race conditions
2. **Memory Efficiency**: Handle large datasets without memory exhaustion  
3. **Modular Design**: Clear separation of concerns with focused packages
4. **Error Transparency**: Structured error handling with full context
5. **Concurrent Safety**: Thread-safe operations throughout the system

## Recent Improvements

### 🔒 Security Enhancements
- **Fixed Critical Vulnerability**: Removed dangerous `os.Setenv` calls in authentication
- **Context-Based Auth**: Secure, request-scoped authentication without global state
- **Race Condition Prevention**: Thread-safe authentication flow

### 📦 Modular Architecture
- **Package Restructuring**: Broke down 1000+ line monolithic main package
- **Focused Modules**: Clear separation between server, loader, auth, and memory packages
- **Dependency Management**: Clean module boundaries with minimal coupling

### 🚀 Performance Optimizations
- **Memory Pools**: Reusable buffer management for large operations
- **Streaming Processing**: Handle large JSON datasets without memory exhaustion
- **Spec Optimization**: Compress and optimize OpenAPI specifications

### 🛠 Developer Experience
- **Structured Errors**: Typed error system with context and stack traces
- **Comprehensive Logging**: Detailed operation tracking and debugging
- **Clear Interfaces**: Well-defined APIs between components

## Viewing Mermaid Diagrams

The Mermaid diagrams in these docs can be viewed in:
- **GitHub** (native support)
- **VS Code** with Mermaid extension
- **Mermaid Live Editor** (https://mermaid.live/)
- **Any Markdown viewer** with Mermaid support

## Contributing

When modifying the architecture:
1. Update relevant documentation files
2. Regenerate Mermaid diagrams if structure changes
3. Update code examples to reflect current implementation
4. Test all architectural components together

---

*This documentation reflects the post-refactoring architecture with security fixes and performance optimizations.*