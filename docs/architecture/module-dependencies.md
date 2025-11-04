# Module Dependencies Architecture

This document outlines the modular structure and dependency relationships in the refactored OpenAPI MCP system, showing how the monolithic main package was broken down into focused, reusable modules.

## Package Structure Overview

```mermaid
graph TB
    subgraph "Application Layer"
        Main[main.go]
        CMD[cmd/]
    end
    
    subgraph "Core Packages"
        Server[pkg/server/]
        Auth[pkg/auth/]
        Loader[pkg/loader/]
        Memory[pkg/memory/]
    end
    
    subgraph "Service Layer"
        Services[pkg/services/]
        OpenAPI2MCP[pkg/openapi2mcp/]
        MCP[pkg/mcp/]
    end
    
    subgraph "Data Layer"
        Database[pkg/database/]
        Models[pkg/models/]
    end
    
    subgraph "External Dependencies"
        OpenAPI3[github.com/getkin/kin-openapi]
        PostgreSQL[github.com/lib/pq]
        HTTP[net/http]
    end
    
    Main --> Server
    Main --> Auth
    Main --> Loader
    Main --> Services
    
    Server --> Auth
    Server --> Memory
    Loader --> Services
    Loader --> Auth
    Loader --> Memory
    
    Services --> Database
    Services --> Models
    OpenAPI2MCP --> Models
    
    Database --> PostgreSQL
    Auth --> OpenAPI3
    Loader --> OpenAPI3
```

## Module Responsibilities

### 1. `pkg/server/` - HTTP Server Components
**Purpose**: HTTP request handling, configuration, and response management

```mermaid
graph TB
    subgraph "pkg/server/"
        Handler[handler.go]
        Config[config.go]
        Errors[errors.go]
        
        subgraph "Responsibilities"
            ReqHandling[Request Handling]
            ConfigMgmt[Configuration Management]
            ErrorHandling[Structured Error Handling]
            ResponseFormatting[Response Formatting]
        end
        
        Handler --> ReqHandling
        Handler --> ResponseFormatting
        Config --> ConfigMgmt
        Errors --> ErrorHandling
    end
    
    subgraph "Dependencies"
        AuthPkg[pkg/auth/]
        ModelsPkg[pkg/models/]
        Context[context]
        HTTP[net/http]
    end
    
    Handler --> AuthPkg
    Handler --> ModelsPkg
    Config --> HTTP
    Errors --> Context
```

**Key Components**:
- **handler.go**: HTTP endpoint handlers (health, reload, API list)
- **config.go**: Server configuration management
- **errors.go**: Structured error types with context

### 2. `pkg/auth/` - Authentication System
**Purpose**: Secure, context-based authentication without global state

```mermaid
graph TB
    subgraph "pkg/auth/"
        Context[context.go]
        Secure[secure.go]
        StateManager[state_manager.go]
        MCPWrapper[mcp_wrapper.go]
        
        subgraph "Security Features"
            ContextAuth[Context-Based Auth]
            NoGlobalState[No Global State Mutation]
            ThreadSafe[Thread-Safe Operations]
            RequestScoped[Request-Scoped Tokens]
        end
        
        Context --> ContextAuth
        Context --> RequestScoped
        Secure --> NoGlobalState
        Secure --> ThreadSafe
        StateManager --> ThreadSafe
    end
    
    subgraph "Dependencies"
        OpenAPI3[github.com/getkin/kin-openapi]
        ModelsPkg[pkg/models/]
        NetHTTP[net/http]
        SyncPkg[sync]
    end
    
    Context --> OpenAPI3
    Context --> ModelsPkg
    Secure --> NetHTTP
    StateManager --> SyncPkg
```

**Key Components**:
- **context.go**: Authentication context creation and management
- **secure.go**: Secure authentication providers and request modification
- **state_manager.go**: Thread-safe spec-to-auth mapping
- **mcp_wrapper.go**: HTTP client wrappers for secure requests

### 3. `pkg/loader/` - Specification Loading
**Purpose**: Memory-efficient OpenAPI specification loading and management

```mermaid
graph TB
    subgraph "pkg/loader/"
        SpecLoader[spec_loader.go]
        
        subgraph "Loading Strategies"
            DatabaseLoad[Database Loading]
            FileLoad[File Loading]
            URLLoad[URL Loading]
            MemoryOptimized[Memory Optimized Processing]
        end
        
        SpecLoader --> DatabaseLoad
        SpecLoader --> FileLoad
        SpecLoader --> URLLoad
        SpecLoader --> MemoryOptimized
    end
    
    subgraph "Dependencies"
        ServicesPkg[pkg/services/]
        AuthPkg[pkg/auth/]
        MemoryPkg[pkg/memory/]
        ServerPkg[pkg/server/]
        OpenAPI3[github.com/getkin/kin-openapi]
    end
    
    SpecLoader --> ServicesPkg
    SpecLoader --> AuthPkg
    SpecLoader --> MemoryPkg
    SpecLoader --> ServerPkg
    SpecLoader --> OpenAPI3
```

**Key Components**:
- **spec_loader.go**: Main specification loading logic with memory optimization
- Integration with memory management for large specs
- Support for multiple loading sources (DB, files, URLs)

### 4. `pkg/memory/` - Memory Optimization
**Purpose**: Memory-efficient processing for large datasets and specifications

```mermaid
graph TB
    subgraph "pkg/memory/"
        Pool[pool.go]
        Streaming[streaming.go]
        OpenAPIOptim[openapi.go]
        
        subgraph "Optimization Techniques"
            BufferPooling[Buffer Pooling]
            StreamProcessing[Stream Processing]
            MemoryLimiting[Memory Limiting]
            SpecOptimization[Spec Optimization]
        end
        
        Pool --> BufferPooling
        Pool --> MemoryLimiting
        Streaming --> StreamProcessing
        OpenAPIOptim --> SpecOptimization
    end
    
    subgraph "Dependencies"
        Runtime[runtime]
        IO[io]
        JSON[encoding/json]
        OpenAPI3[github.com/getkin/kin-openapi]
        SyncPkg[sync]
    end
    
    Pool --> Runtime
    Pool --> SyncPkg
    Streaming --> IO
    Streaming --> JSON
    OpenAPIOptim --> OpenAPI3
```

**Key Components**:
- **pool.go**: Buffer pools and memory limiters
- **streaming.go**: Streaming JSON processing and chunked writing
- **openapi.go**: OpenAPI specification memory optimization

## Dependency Flow Analysis

### Before Refactoring (Monolithic)
```mermaid
graph TB
    subgraph "Monolithic main.go (1000+ lines)"
        MainMono[main.go]
        
        subgraph "Mixed Responsibilities"
            HTTPHandling[HTTP Handling]
            AuthLogic[Authentication Logic]
            SpecLoading[Spec Loading]
            ErrorHandling[Error Handling]
            ConfigMgmt[Configuration]
            MemoryMgmt[Memory Management]
        end
        
        MainMono --> HTTPHandling
        MainMono --> AuthLogic
        MainMono --> SpecLoading
        MainMono --> ErrorHandling
        MainMono --> ConfigMgmt
        MainMono --> MemoryMgmt
        
        subgraph "Issues"
            Coupling[High Coupling]
            TestDifficulty[Hard to Test]
            Maintenance[Hard to Maintain]
            Security[Security Vulnerabilities]
        end
        
        HTTPHandling -.-> Coupling
        AuthLogic -.-> Security
        MainMono -.-> TestDifficulty
        MainMono -.-> Maintenance
    end
```

### After Refactoring (Modular)
```mermaid
graph TB
    subgraph "Modular Architecture"
        MainNew[main.go (simplified)]
        
        subgraph "Focused Packages"
            ServerPkg[pkg/server/ - HTTP & Config]
            AuthPkg[pkg/auth/ - Security]
            LoaderPkg[pkg/loader/ - Spec Management]
            MemoryPkg[pkg/memory/ - Optimization]
        end
        
        MainNew --> ServerPkg
        MainNew --> AuthPkg
        MainNew --> LoaderPkg
        
        ServerPkg --> AuthPkg
        LoaderPkg --> AuthPkg
        LoaderPkg --> MemoryPkg
        
        subgraph "Benefits"
            LowCoupling[Low Coupling]
            Testable[Easily Testable]
            Maintainable[Maintainable]
            Secure[Security Fixed]
        end
        
        ServerPkg -.-> LowCoupling
        AuthPkg -.-> Secure
        MainNew -.-> Testable
        MainNew -.-> Maintainable
    end
```

## Package Interface Contracts

### 1. Authentication Interface
```go
// pkg/auth/secure.go
type SecureAuthProvider interface {
    GetAuthHeaders(ctx context.Context) map[string]string
    GetAuthQueryParams(ctx context.Context) map[string]string
}

// pkg/auth/context.go
func CreateAuthContext(r *http.Request, doc *openapi3.T, spec *models.OpenAPISpec) *AuthContext
func WithAuthContext(ctx context.Context, authCtx *AuthContext) context.Context
func FromContext(ctx context.Context) (*AuthContext, bool)
```

### 2. Memory Management Interface
```go
// pkg/memory/pool.go
type MemoryLimiter interface {
    CheckMemoryUsage() bool
    GetMemoryStats() (allocMB, sysMB int64)
}

// pkg/memory/streaming.go
type StreamingJSONProcessor interface {
    ProcessLargeJSON(ctx context.Context, reader io.Reader, callback func(interface{}) error) error
}
```

### 3. Server Configuration Interface
```go
// pkg/server/config.go
type Config struct {
    DatabaseMode bool
    HTTPMode     bool
    HTTPAddr     string
    DatabaseURL  string
    Port         int
    SpecFiles    []string
}

func LoadConfig(args []string) (*Config, error)
func (c *Config) Validate() error
```

## Dependency Injection Pattern

### Service Initialization Flow
```mermaid
sequenceDiagram
    participant Main as main.go
    participant Config as pkg/server/Config
    participant Auth as pkg/auth/StateManager
    participant Memory as pkg/memory/StreamProcessor
    participant Loader as pkg/loader/SpecLoader
    participant Server as HTTP Server
    
    Main->>Config: LoadConfig(args)
    Config-->>Main: Server configuration
    
    Main->>Auth: NewStateManager()
    Auth-->>Main: Auth state manager
    
    Main->>Memory: NewStreamProcessor(limits)
    Memory-->>Main: Memory processor
    
    Main->>Loader: NewSpecLoader(services, auth)
    Loader-->>Main: Spec loader
    
    Main->>Server: Configure with all components
    Server-->>Main: HTTP server ready
```

### Component Lifecycle Management
```go
// Dependency injection in main.go
func initializeComponents(config *serverPkg.Config) (*Components, error) {
    // Initialize core services
    authStateManager := auth.NewStateManager()
    memoryProcessor := memory.NewStreamProcessor(config.MaxMemoryMB, config.ChunkSize)
    
    // Initialize spec loader with dependencies
    specLoader := loader.NewSpecLoader(specLoaderService, authStateManager)
    
    // Create server with all dependencies
    server := createServerWithDependencies(config, authStateManager, specLoader, memoryProcessor)
    
    return &Components{
        AuthManager:     authStateManager,
        MemoryProcessor: memoryProcessor,
        SpecLoader:      specLoader,
        Server:          server,
    }, nil
}
```

## Testing Architecture

### Package-Level Testing
```mermaid
graph TB
    subgraph "Testing Strategy"
        subgraph "Unit Tests"
            AuthTests[pkg/auth/*_test.go]
            ServerTests[pkg/server/*_test.go]
            MemoryTests[pkg/memory/*_test.go]
            LoaderTests[pkg/loader/*_test.go]
        end
        
        subgraph "Integration Tests"
            APITests[API Integration Tests]
            DatabaseTests[Database Integration Tests]
            SecurityTests[Security Integration Tests]
        end
        
        subgraph "End-to-End Tests"
            E2ETests[Full System Tests]
        end
    end
    
    AuthTests --> APITests
    ServerTests --> APITests
    MemoryTests --> APITests
    LoaderTests --> DatabaseTests
    
    APITests --> E2ETests
    DatabaseTests --> E2ETests
    SecurityTests --> E2ETests
```

### Testable Component Design
Each package is designed with testing in mind:

```go
// Example: Testable authentication component
func TestSecureAuthProvider(t *testing.T) {
    // Create test context with auth
    ctx := context.Background()
    authCtx := &auth.AuthContext{
        Token:    "test-token",
        AuthType: "bearer",
        Endpoint: "test-api",
    }
    ctx = auth.WithAuthContext(ctx, authCtx)
    
    // Test provider
    provider := auth.NewSecureAuthProvider()
    headers := provider.GetAuthHeaders(ctx)
    
    assert.Equal(t, "Bearer test-token", headers["Authorization"])
}
```

## Migration Path

### From Monolithic to Modular
1. **Extract Interfaces**: Define clear interfaces for each component
2. **Move Code**: Relocate code to appropriate packages
3. **Update Imports**: Change import statements to use new packages
4. **Inject Dependencies**: Pass components as dependencies rather than globals
5. **Test Integration**: Ensure all components work together

### Backward Compatibility
The refactored architecture maintains API compatibility:
- External APIs remain unchanged
- Configuration format is preserved
- Database schema is unmodified
- MCP protocol compliance is maintained

---

*This modular architecture provides clear separation of concerns, improved testability, and better maintainability while eliminating security vulnerabilities.*