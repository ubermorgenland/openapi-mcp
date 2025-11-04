# Error Handling Architecture

The OpenAPI MCP system implements a comprehensive structured error handling system that provides detailed context, tracing, and proper categorization for all error conditions.

## Structured Error System Overview

```mermaid
graph TB
    subgraph "Error Handling Architecture"
        subgraph "Error Sources"
            APIErrors[API Call Errors]
            DatabaseErrors[Database Errors]
            ValidationErrors[Validation Errors]
            AuthErrors[Authentication Errors]
            NetworkErrors[Network Errors]
            InternalErrors[Internal Errors]
        end
        
        subgraph "Error Processing"
            Wrapper[Error Wrapper]
            Context[Error Context]
            Logger[Error Logger]
            ResponseFormatter[Response Formatter]
        end
        
        subgraph "Error Output"
            StructuredLog[Structured Logs]
            HTTPResponse[HTTP Error Response]
            Metrics[Error Metrics]
            Alerts[Error Alerts]
        end
        
        APIErrors --> Wrapper
        DatabaseErrors --> Wrapper
        ValidationErrors --> Wrapper
        AuthErrors --> Wrapper
        NetworkErrors --> Wrapper
        InternalErrors --> Wrapper
        
        Wrapper --> Context
        Context --> Logger
        Context --> ResponseFormatter
        
        Logger --> StructuredLog
        ResponseFormatter --> HTTPResponse
        Logger --> Metrics
        Metrics --> Alerts
    end
```

## Error Type System

### Error Type Definitions
**Location**: `pkg/server/errors.go`

```go
type ErrorType string

const (
    ErrorTypeValidation   ErrorType = "validation"
    ErrorTypeDatabase     ErrorType = "database"
    ErrorTypeAuth         ErrorType = "authentication"
    ErrorTypeNetwork      ErrorType = "network"
    ErrorTypeInternal     ErrorType = "internal"
    ErrorTypeNotFound     ErrorType = "not_found"
    ErrorTypeConflict     ErrorType = "conflict"
)
```

### Structured Error Format

```mermaid
graph LR
    subgraph "ServerError Structure"
        Type[Error Type]
        Message[Error Message]
        Details[Error Details]
        RequestID[Request ID]
        Timestamp[Timestamp]
        StackTrace[Stack Trace]
        
        subgraph "Context Information"
            UserID[User ID]
            Operation[Operation Context]
            Resources[Affected Resources]
        end
        
        Type --> ServerError[ServerError]
        Message --> ServerError
        Details --> ServerError
        RequestID --> ServerError
        Timestamp --> ServerError
        StackTrace --> ServerError
        
        UserID --> ServerError
        Operation --> ServerError
        Resources --> ServerError
    end
```

```go
type ServerError struct {
    Type       ErrorType `json:"type"`
    Message    string    `json:"message"`
    Details    string    `json:"details,omitempty"`
    RequestID  string    `json:"request_id,omitempty"`
    Timestamp  int64     `json:"timestamp"`
    StackTrace string    `json:"stack_trace,omitempty"`
}

func (e *ServerError) Error() string {
    if e.Details != "" {
        return fmt.Sprintf("%s: %s (%s)", e.Type, e.Message, e.Details)
    }
    return fmt.Sprintf("%s: %s", e.Type, e.Message)
}
```

## Error Creation and Wrapping

### Error Creation Flow
```mermaid
sequenceDiagram
    participant Source as Error Source
    participant Wrapper as Error Wrapper
    participant Context as Request Context
    participant Logger as Error Logger
    participant Response as HTTP Response
    
    Source->>Wrapper: Original Error
    Context->>Wrapper: Request Context
    Wrapper->>Wrapper: Create ServerError
    Wrapper->>Wrapper: Add context information
    Wrapper->>Logger: Log structured error
    Wrapper->>Response: Format error response
    
    Note over Wrapper: Enriches error with context
    Note over Logger: Structured logging with metadata
```

### Error Creation Methods

#### Creating New Errors
```go
// Create new error with context
func NewErrorWithContext(ctx context.Context, errType ErrorType, message string, details string) *ServerError {
    err := NewError(errType, message, details)
    
    // Extract request ID from context if available
    if requestID, ok := ctx.Value("request_id").(string); ok {
        err.RequestID = requestID
    }
    
    return err
}

// Usage example
func validateAPISpec(ctx context.Context, spec []byte) error {
    if len(spec) == 0 {
        return NewErrorWithContext(ctx, ErrorTypeValidation, 
            "empty API specification", "spec content is required")
    }
    
    // Validation logic...
    return nil
}
```

#### Wrapping Existing Errors
```go
// Wrap existing error with context
func WrapWithContext(ctx context.Context, err error, errType ErrorType, message string) *ServerError {
    if err == nil {
        return nil
    }
    
    return NewErrorWithContext(ctx, errType, message, err.Error())
}

// Usage example
func loadSpecFromDatabase(ctx context.Context, id int) (*Spec, error) {
    spec, err := db.GetSpec(id)
    if err != nil {
        return nil, WrapWithContext(ctx, err, ErrorTypeDatabase, 
            "failed to load API specification")
    }
    
    return spec, nil
}
```

## Error Logging Architecture

### Contextual Logging System
```mermaid
graph TB
    subgraph "Error Logging Flow"
        Error[ServerError]
        LogLevel[Determine Log Level]
        Enrichment[Add Context]
        Formatting[Format Log Entry]
        Output[Log Output]
        
        subgraph "Log Enrichment"
            RequestInfo[Request Information]
            UserContext[User Context]
            SystemState[System State]
            Performance[Performance Metrics]
        end
        
        Error --> LogLevel
        LogLevel --> Enrichment
        Enrichment --> RequestInfo
        Enrichment --> UserContext
        Enrichment --> SystemState
        Enrichment --> Performance
        Enrichment --> Formatting
        Formatting --> Output
    end
```

### Log Level Determination
```go
func (e *ServerError) LogError() {
    switch e.Type {
    case ErrorTypeValidation:
        log.Printf("VALIDATION ERROR: %s", e.Error())
    case ErrorTypeAuth:
        log.Printf("AUTH ERROR: %s", e.Error())
    case ErrorTypeDatabase:
        log.Printf("DATABASE ERROR: %s", e.Error())
    case ErrorTypeNetwork:
        log.Printf("NETWORK ERROR: %s", e.Error())
    case ErrorTypeNotFound:
        log.Printf("NOT FOUND: %s", e.Error())
    case ErrorTypeConflict:
        log.Printf("CONFLICT: %s", e.Error())
    default:
        log.Printf("ERROR: %s", e.Error())
    }
    
    if e.StackTrace != "" {
        log.Printf("Stack trace: %s", e.StackTrace)
    }
}
```

## Error Response Formatting

### HTTP Error Response Structure
```mermaid
graph LR
    subgraph "HTTP Error Response"
        StatusCode[HTTP Status Code]
        Headers[Error Headers]
        Body[JSON Error Body]
        
        subgraph "JSON Body Structure"
            ErrorField[error]
            TypeField[type]
            MessageField[message]
            DetailsField[details]
            RequestIDField[request_id]
            TimestampField[timestamp]
        end
        
        Body --> ErrorField
        Body --> TypeField
        Body --> MessageField
        Body --> DetailsField
        Body --> RequestIDField
        Body --> TimestampField
    end
```

### Status Code Mapping
```go
func getHTTPStatusCode(errType ErrorType) int {
    switch errType {
    case ErrorTypeValidation:
        return http.StatusBadRequest // 400
    case ErrorTypeAuth:
        return http.StatusUnauthorized // 401
    case ErrorTypeNotFound:
        return http.StatusNotFound // 404
    case ErrorTypeConflict:
        return http.StatusConflict // 409
    case ErrorTypeDatabase:
        return http.StatusInternalServerError // 500
    case ErrorTypeNetwork:
        return http.StatusBadGateway // 502
    default:
        return http.StatusInternalServerError // 500
    }
}
```

### Error Response Example
```json
{
    "error": true,
    "type": "validation",
    "message": "Invalid OpenAPI specification",
    "details": "spec validation failed: missing required field 'openapi'",
    "request_id": "req_123456789",
    "timestamp": 1699123456
}
```

## Error Handling in Different Components

### 1. Authentication Errors
```mermaid
sequenceDiagram
    participant Client
    participant Auth as Auth Module
    participant Context as Auth Context
    participant Error as Error Handler
    participant Response as HTTP Response
    
    Client->>Auth: Request with invalid credentials
    Auth->>Context: Create auth context
    Context->>Context: Validate credentials
    Context->>Error: Create auth error
    Error->>Error: Log security event
    Error->>Response: 401 Unauthorized
    Response->>Client: Error response
    
    Note over Error: Security-focused logging
    Note over Response: No sensitive info leaked
```

```go
func validateAuthToken(ctx context.Context, token string) error {
    if token == "" {
        return NewErrorWithContext(ctx, ErrorTypeAuth, 
            "authentication required", "no authentication token provided")
    }
    
    if !isValidToken(token) {
        return NewErrorWithContext(ctx, ErrorTypeAuth, 
            "invalid authentication token", "token validation failed")
    }
    
    return nil
}
```

### 2. Database Errors
```mermaid
graph TB
    subgraph "Database Error Handling"
        DBOp[Database Operation]
        DBError[Database Error]
        Classification[Error Classification]
        Context[Add Context]
        Recovery[Recovery Attempt]
        Logging[Structured Logging]
        Response[Client Response]
        
        DBOp --> DBError
        DBError --> Classification
        Classification --> Context
        Context --> Recovery
        Recovery --> Logging
        Logging --> Response
        
        subgraph "Error Classifications"
            Connection[Connection Error]
            Timeout[Timeout Error]
            Constraint[Constraint Violation]
            NotFound[Record Not Found]
        end
        
        Classification --> Connection
        Classification --> Timeout
        Classification --> Constraint
        Classification --> NotFound
    end
```

```go
func getSpecFromDatabase(ctx context.Context, id int) (*models.OpenAPISpec, error) {
    spec, err := db.GetSpec(id)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, NewErrorWithContext(ctx, ErrorTypeNotFound, 
                "API specification not found", fmt.Sprintf("spec ID %d does not exist", id))
        }
        
        return nil, WrapWithContext(ctx, err, ErrorTypeDatabase, 
            "database query failed")
    }
    
    return spec, nil
}
```

### 3. Validation Errors
```go
func validateOpenAPISpec(ctx context.Context, specContent []byte) error {
    if len(specContent) == 0 {
        return NewErrorWithContext(ctx, ErrorTypeValidation,
            "empty specification", "specification content cannot be empty")
    }
    
    loader := openapi3.NewLoader()
    doc, err := loader.LoadFromData(specContent)
    if err != nil {
        return WrapWithContext(ctx, err, ErrorTypeValidation,
            "invalid OpenAPI specification format")
    }
    
    if err := doc.Validate(ctx); err != nil {
        return WrapWithContext(ctx, err, ErrorTypeValidation,
            "OpenAPI specification validation failed")
    }
    
    return nil
}
```

## Error Propagation and Context

### Context Propagation Pattern
```mermaid
graph TB
    subgraph "Error Context Propagation"
        Request[HTTP Request]
        Context[Request Context]
        Service1[Service Layer 1]
        Service2[Service Layer 2]
        Service3[Service Layer 3]
        ErrorResponse[Error Response]
        
        Request --> Context
        Context --> Service1
        Service1 --> Service2
        Service2 --> Service3
        Service3 --> ErrorResponse
        
        subgraph "Context Information"
            RequestID[Request ID]
            UserInfo[User Information]
            Operation[Operation Context]
            Timing[Timing Information]
        end
        
        Context --> RequestID
        Context --> UserInfo
        Context --> Operation
        Context --> Timing
        
        RequestID -.-> ErrorResponse
        UserInfo -.-> ErrorResponse
        Operation -.-> ErrorResponse
        Timing -.-> ErrorResponse
    end
```

### Context-Aware Error Handling
```go
// Create request context with ID
func createRequestContext(r *http.Request) context.Context {
    requestID := generateRequestID()
    ctx := context.WithValue(r.Context(), "request_id", requestID)
    ctx = context.WithValue(ctx, "start_time", time.Now())
    return ctx
}

// Use context in error creation
func processAPIRequest(ctx context.Context, spec *models.OpenAPISpec) error {
    if err := validateSpec(ctx, spec); err != nil {
        return err // Context already included
    }
    
    if err := loadSpecToCache(ctx, spec); err != nil {
        return WrapWithContext(ctx, err, ErrorTypeInternal,
            "failed to cache specification")
    }
    
    return nil
}
```

## Error Recovery Strategies

### Retry Mechanisms
```mermaid
graph TB
    subgraph "Error Recovery Flow"
        Operation[Original Operation]
        Error[Error Occurs]
        Classification[Classify Error]
        RetryDecision{Retryable?}
        Retry[Retry Operation]
        Backoff[Exponential Backoff]
        MaxRetries{Max Retries?}
        Failure[Permanent Failure]
        Success[Operation Success]
        
        Operation --> Error
        Error --> Classification
        Classification --> RetryDecision
        RetryDecision -->|Yes| Retry
        RetryDecision -->|No| Failure
        Retry --> Backoff
        Backoff --> MaxRetries
        MaxRetries -->|No| Retry
        MaxRetries -->|Yes| Failure
        Retry -->|Success| Success
    end
```

### Circuit Breaker Pattern
```go
type CircuitBreaker struct {
    failures    int
    maxFailures int
    timeout     time.Duration
    lastFailure time.Time
    state       string // "closed", "open", "half-open"
}

func (cb *CircuitBreaker) Call(ctx context.Context, operation func() error) error {
    if cb.state == "open" {
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.state = "half-open"
        } else {
            return NewErrorWithContext(ctx, ErrorTypeNetwork,
                "circuit breaker open", "service temporarily unavailable")
        }
    }
    
    err := operation()
    if err != nil {
        cb.failures++
        cb.lastFailure = time.Now()
        if cb.failures >= cb.maxFailures {
            cb.state = "open"
        }
        return WrapWithContext(ctx, err, ErrorTypeNetwork, "operation failed")
    }
    
    cb.failures = 0
    cb.state = "closed"
    return nil
}
```

## Error Monitoring and Alerting

### Error Metrics Collection
```mermaid
graph LR
    subgraph "Error Monitoring Pipeline"
        Errors[Application Errors]
        Collector[Metrics Collector]
        
        subgraph "Metrics"
            ErrorCount[Error Count by Type]
            ErrorRate[Error Rate]
            ResponseTime[Response Time]
            SuccessRate[Success Rate]
        end
        
        subgraph "Alerting"
            Thresholds[Alert Thresholds]
            Notifications[Notifications]
            Dashboard[Error Dashboard]
        end
        
        Errors --> Collector
        Collector --> ErrorCount
        Collector --> ErrorRate
        Collector --> ResponseTime
        Collector --> SuccessRate
        
        ErrorCount --> Thresholds
        ErrorRate --> Thresholds
        Thresholds --> Notifications
        ErrorCount --> Dashboard
        ErrorRate --> Dashboard
    end
```

### Error Analysis
- **Error Patterns**: Identify common error patterns and root causes
- **Performance Impact**: Measure error impact on system performance
- **User Experience**: Track error impact on user operations
- **Trend Analysis**: Monitor error trends over time

---

*This structured error handling architecture provides comprehensive error management with proper context, logging, and recovery strategies for production-ready operations.*