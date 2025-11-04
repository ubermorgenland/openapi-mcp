package server

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ubermorgenland/openapi-mcp/pkg/mcp/mcp"
	"github.com/ubermorgenland/openapi-mcp/pkg/mcp/util"
)

// StreamableHTTPOption defines a function type for configuring StreamableHTTPServer
type StreamableHTTPOption func(*StreamableHTTPServer)

// WithEndpointPath sets the endpoint path for the server.
// The default is "/mcp".
// It's only works for `Start` method. When used as a http.Handler, it has no effect.
func WithEndpointPath(endpointPath string) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		// Normalize the endpoint path to ensure it starts with a slash and doesn't end with one
		normalizedPath := "/" + strings.Trim(endpointPath, "/")
		s.endpointPath = normalizedPath
	}
}

// WithStateLess sets the server to stateless mode.
// If true, the server will manage no session information. Every request will be treated
// as a new session. No session id returned to the client.
// The default is false.
//
// Notice: This is a convenience method. It's identical to set WithSessionIdManager option
// to StatelessSessionIdManager.
func WithStateLess(stateLess bool) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		s.sessionIdManager = &StatelessSessionIdManager{}
	}
}

// WithSessionIdManager sets a custom session id generator for the server.
// By default, the server will use SimpleStatefulSessionIdGenerator, which generates
// session ids with uuid, and it's insecure.
// Notice: it will override the WithStateLess option.
func WithSessionIdManager(manager SessionIdManager) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		s.sessionIdManager = manager
	}
}

// WithHeartbeatInterval sets the heartbeat interval. Positive interval means the
// server will send a heartbeat to the client through the GET connection, to keep
// the connection alive from being closed by the network infrastructure (e.g.
// gateways). If the client does not establish a GET connection, it has no
// effect. The default is not to send heartbeats.
func WithHeartbeatInterval(interval time.Duration) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		s.listenHeartbeatInterval = interval
	}
}

// WithHTTPContextFunc sets a function that will be called to customise the context
// to the server using the incoming request.
// This can be used to inject context values from headers, for example.
func WithHTTPContextFunc(fn HTTPContextFunc) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		s.contextFunc = fn
	}
}

// WithLogger sets the logger for the server
func WithLogger(logger util.Logger) StreamableHTTPOption {
	return func(s *StreamableHTTPServer) {
		s.logger = logger
	}
}

// StreamableHTTPServer implements a Streamable-http based MCP server.
// It communicates with clients over HTTP protocol, supporting both direct HTTP responses, and SSE streams.
// https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#streamable-http
//
// Usage:
//
//	server := NewStreamableHTTPServer(mcpServer)
//	server.Start(":8080") // The final url for client is http://xxxx:8080/mcp by default
//
// or the server itself can be used as a http.Handler, which is convenient to
// integrate with existing http servers, or advanced usage:
//
//	handler := NewStreamableHTTPServer(mcpServer)
//	http.Handle("/streamable-http", handler)
//	http.ListenAndServe(":8080", nil)
//
// Notice:
// Except for the GET handlers(listening), the POST handlers(request/notification) will
// not trigger the session registration. So the methods like `SendNotificationToSpecificClient`
// or `hooks.onRegisterSession` will not be triggered for POST messages.
//
// The current implementation does not support the following features from the specification:
//   - Batching of requests/notifications/responses in arrays.
//   - Stream Resumability
type StreamableHTTPServer struct {
	server       *MCPServer
	sessionTools *sessionToolsStore

	httpServer *http.Server
	mu         sync.RWMutex

	endpointPath            string
	contextFunc             HTTPContextFunc
	sessionIdManager        SessionIdManager
	listenHeartbeatInterval time.Duration
	logger                  util.Logger
	
	// Session cleanup
	cleanupCtx    context.Context
	cleanupCancel context.CancelFunc
	cleanupDone   chan struct{}
}

// NewStreamableHTTPServer creates a new streamable-http server instance
func NewStreamableHTTPServer(server *MCPServer, opts ...StreamableHTTPOption) *StreamableHTTPServer {
	ctx, cancel := context.WithCancel(context.Background())
	s := &StreamableHTTPServer{
		server:           server,
		sessionTools:     newSessionToolsStore(),
		endpointPath:     "/mcp",
		sessionIdManager: &InsecureStatefulSessionIdManager{},
		logger:           util.DefaultLogger(),
		cleanupCtx:       ctx,
		cleanupCancel:    cancel,
		cleanupDone:      make(chan struct{}),
	}

	// Apply all options
	for _, opt := range opts {
		opt(s)
	}
	
	// Start cleanup goroutine
	go s.runSessionCleanup()
	
	return s
}

// ServeHTTP implements the http.Handler interface.
func (s *StreamableHTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Always log incoming requests for debugging
	// TODO: Make this configurable for production
	s.logIncomingRequest(r)
	// Check for optimized API endpoints first
	if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/tools") {
		// Only use optimized API for direct tools endpoint access (not sub-paths like /tools/call)
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) >= 1 && pathParts[len(pathParts)-1] == "tools" {
			s.handleToolsAPI(w, r)
			return
		}
	}
	
	switch r.Method {
	case http.MethodPost:
		s.handlePost(w, r)
	case http.MethodGet:
		s.handleGet(w, r)
	case http.MethodDelete:
		s.handleDelete(w, r)
	default:
		http.NotFound(w, r)
	}
}

// Start begins serving the http server on the specified address and path
// (endpointPath). like:
//
//	s.Start(":8080")
func (s *StreamableHTTPServer) Start(addr string) error {
	s.mu.Lock()
	mux := http.NewServeMux()
	mux.Handle(s.endpointPath, s)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	s.mu.Unlock()

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server, closing all active sessions
// and shutting down the HTTP server.
func (s *StreamableHTTPServer) Shutdown(ctx context.Context) error {
	// Stop cleanup goroutine if it exists
	if s.cleanupCancel != nil {
		s.cleanupCancel()
		
		// Wait for cleanup to finish with timeout
		select {
		case <-s.cleanupDone:
			s.logger.Infof("Session cleanup goroutine stopped")
		case <-time.After(5 * time.Second):
			s.logger.Infof("Session cleanup goroutine stop timeout")
		case <-ctx.Done():
			s.logger.Infof("Session cleanup stopped due to context cancellation")
		}
	}
	
	// shutdown the server if needed (may use as a http.Handler)
	s.mu.RLock()
	srv := s.httpServer
	s.mu.RUnlock()
	if srv != nil {
		return srv.Shutdown(ctx)
	}
	return nil
}

// --- internal methods ---

const (
	headerKeySessionID = "Mcp-Session-Id"
)

// extractAuthHeaders extracts authentication-related headers from the HTTP request
func extractAuthHeaders(headers http.Header) http.Header {
	authHeaders := make(http.Header)
	
	// List of header keys that should be preserved for authentication
	authHeaderKeys := []string{
		"Authorization",
		"X-API-Key", 
		"X-Auth-Token",
		"Bearer",
		"Token",
		"API-Key",
		"Auth-Token",
	}
	
	for _, key := range authHeaderKeys {
		if values := headers.Values(key); len(values) > 0 {
			authHeaders[key] = values
		}
	}
	
	return authHeaders
}

func (s *StreamableHTTPServer) handlePost(w http.ResponseWriter, r *http.Request) {
	// post request carry request/notification message

	// Check content type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(w, "Invalid content type: must be 'application/json'", http.StatusBadRequest)
		return
	}

	// Check the request body is valid json, meanwhile, get the request Method
	rawData, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeJSONRPCError(w, nil, mcp.PARSE_ERROR, fmt.Sprintf("read request body error: %v", err))
		return
	}
	var baseMessage struct {
		Method mcp.MCPMethod `json:"method"`
	}
	if err := json.Unmarshal(rawData, &baseMessage); err != nil {
		s.writeJSONRPCError(w, nil, mcp.PARSE_ERROR, "request body is not valid json")
		return
	}
	isInitializeRequest := baseMessage.Method == mcp.MethodInitialize

	// Prepare the session for the mcp server
	// The session is ephemeral. Its life is the same as the request. It's only created
	// for interaction with the mcp server.
	var sessionID string
	if isInitializeRequest {
		// generate a new one for initialize request
		sessionID = s.sessionIdManager.Generate()
	} else {
		// Get session ID from header.
		// Stateful servers need the client to carry the session ID.
		sessionID = r.Header.Get(headerKeySessionID)
		isTerminated, err := s.sessionIdManager.Validate(sessionID)
		if err != nil {
			http.Error(w, "Invalid session ID", http.StatusBadRequest)
			return
		}
		if isTerminated {
			http.Error(w, "Session terminated", http.StatusNotFound)
			return
		}
		
		// Touch session to renew its expiration when accessed
		if sessionID != "" {
			if err := s.server.TouchSession(sessionID, DefaultSessionTimeout); err != nil {
				// Log error but don't fail the request - session might not support expiration
				s.logger.Infof("Failed to touch session %s: %v", sessionID, err)
			}
		}
	}

	// Extract authentication headers from the request
	authHeaders := extractAuthHeaders(r.Header)
	session := newStreamableHttpSessionWithHeaders(sessionID, s.sessionTools, authHeaders)
	
	// Debug: Log extracted headers
	if len(authHeaders) > 0 {
		for key, values := range authHeaders {
			log.Printf("DEBUG: Extracted session auth header %s: %s", key, strings.Join(values, ", "))
		}
	}

	// Register the session with the MCP server so authentication can find it
	sessionRegistered := false
	if sessionID != "" {
		if err := s.server.RegisterSession(r.Context(), session); err != nil {
			// If session already exists, that's fine for streamable HTTP (ephemeral sessions)
			if err != ErrSessionExists {
				s.logger.Errorf("Failed to register session %s: %v", sessionID, err)
			}
		} else {
			sessionRegistered = true
		}
	}
	
	// Clean up session when request is done
	defer func() {
		if sessionRegistered && sessionID != "" {
			s.server.UnregisterSession(context.Background(), sessionID)
		}
	}()

	// Set the client context before handling the message
	ctx := s.server.WithContext(r.Context(), session)
	if s.contextFunc != nil {
		ctx = s.contextFunc(ctx, r)
	}

	// handle potential notifications
	mu := sync.Mutex{}
	upgraded := false
	done := make(chan struct{})
	defer close(done)

	go func() {
		for {
			select {
			case nt := <-session.notificationChannel:
				func() {
					mu.Lock()
					defer mu.Unlock()
					defer func() {
						flusher, ok := w.(http.Flusher)
						if ok {
							flusher.Flush()
						}
					}()

					// if there's notifications, upgrade to SSE response
					if !upgraded {
						upgraded = true
						w.Header().Set("Content-Type", "text/event-stream")
						w.Header().Set("Connection", "keep-alive")
						w.Header().Set("Cache-Control", "no-cache")
						w.WriteHeader(http.StatusAccepted)
					}
					err := writeSSEEvent(w, nt)
					if err != nil {
						s.logger.Errorf("Failed to write SSE event: %v", err)
						return
					}
				}()
			case <-done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Process message through MCPServer
	response := s.server.HandleMessage(ctx, rawData)
	if response == nil {
		// For notifications, just send 202 Accepted with no body
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// Write response
	mu.Lock()
	defer mu.Unlock()
	if ctx.Err() != nil {
		return
	}
	if upgraded {
		if err := writeSSEEvent(w, response); err != nil {
			s.logger.Errorf("Failed to write final SSE response event: %v", err)
		}
	} else {
		// Check if compression should be used for large responses
		shouldCompress := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
		
		if shouldCompress {
			// Marshal response to check size
			responseData, err := json.Marshal(response)
			if err != nil {
				s.logger.Errorf("Failed to marshal response: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			
			// Apply compression if response is larger than 1KB
			if len(responseData) > 1024 {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Set("Vary", "Accept-Encoding")
				if isInitializeRequest && sessionID != "" {
					w.Header().Set(headerKeySessionID, sessionID)
				}
				
				gz := gzip.NewWriter(w)
				defer gz.Close()
				
				w.WriteHeader(http.StatusOK)
				_, err = gz.Write(responseData)
				if err != nil {
					s.logger.Errorf("Compression error: %v", err)
				}
				return
			}
		}
		
		// Fallback to uncompressed response
		w.Header().Set("Content-Type", "application/json")
		if isInitializeRequest && sessionID != "" {
			// send the session ID back to the client
			w.Header().Set(headerKeySessionID, sessionID)
		}
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			s.logger.Errorf("Failed to write response: %v", err)
		}
	}
}

func (s *StreamableHTTPServer) handleGet(w http.ResponseWriter, r *http.Request) {
	// get request is for listening to notifications
	// https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#listening-for-messages-from-the-server

	sessionID := r.Header.Get(headerKeySessionID)
	// the specification didn't say we should validate the session id

	if sessionID == "" {
		// It's a stateless server,
		// but the MCP server requires a unique ID for registering, so we use a random one
		sessionID = uuid.New().String()
	}

	session := newStreamableHttpSession(sessionID, s.sessionTools)
	if err := s.server.RegisterSession(r.Context(), session); err != nil {
		http.Error(w, fmt.Sprintf("Session registration failed: %v", err), http.StatusBadRequest)
		return
	}
	defer s.server.UnregisterSession(r.Context(), sessionID)

	// Set the client context before handling the message
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusAccepted)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}
	flusher.Flush()

	// Send initial endpoint event with session information
	endpointData := fmt.Sprintf("?sessionId=%s", sessionID)
	if err := writeSSEEventWithType(w, "endpoint", endpointData); err != nil {
		s.logger.Errorf("Failed to write initial endpoint event: %v", err)
		return
	}
	flusher.Flush()

	// Start notification handler for this session
	done := make(chan struct{})
	defer close(done)
	writeChan := make(chan any, 16)

	go func() {
		for {
			select {
			case nt := <-session.notificationChannel:
				select {
				case writeChan <- &nt:
				case <-done:
					return
				}
			case <-done:
				return
			}
		}
	}()

	if s.listenHeartbeatInterval > 0 {
		// heartbeat to keep the connection alive
		go func() {
			ticker := time.NewTicker(s.listenHeartbeatInterval)
			defer ticker.Stop()
			message := mcp.JSONRPCRequest{
				JSONRPC: "2.0",
				Request: mcp.Request{
					Method: "ping",
				},
			}
			for {
				select {
				case <-ticker.C:
					select {
					case writeChan <- message:
					case <-done:
						return
					}
				case <-done:
					return
				}
			}
		}()
	}

	// Keep the connection open until the client disconnects
	//
	// There's will a Available() check when handler ends, and it maybe race with Flush(),
	// so we use a separate channel to send the data, inteading of flushing directly in other goroutine.
	for {
		select {
		case data := <-writeChan:
			if data == nil {
				continue
			}
			if err := writeSSEEvent(w, data); err != nil {
				s.logger.Errorf("Failed to write SSE event: %v", err)
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *StreamableHTTPServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	// delete request terminate the session
	sessionID := r.Header.Get(headerKeySessionID)
	notAllowed, err := s.sessionIdManager.Terminate(sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Session termination failed: %v", err), http.StatusInternalServerError)
		return
	}
	if notAllowed {
		http.Error(w, "Session termination not allowed", http.StatusMethodNotAllowed)
		return
	}

	// remove the session relateddata from the sessionToolsStore
	s.sessionTools.set(sessionID, nil)

	w.WriteHeader(http.StatusOK)
}

func writeSSEEvent(w io.Writer, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	_, err = fmt.Fprintf(w, "event: message\ndata: %s\n\n", jsonData)
	if err != nil {
		return fmt.Errorf("failed to write SSE event: %w", err)
	}
	return nil
}

func writeSSEEventWithType(w io.Writer, eventType string, data any) error {
	var dataStr string
	switch v := data.(type) {
	case string:
		dataStr = v
	default:
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal data: %w", err)
		}
		dataStr = string(jsonData)
	}
	_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, dataStr)
	if err != nil {
		return fmt.Errorf("failed to write SSE event: %w", err)
	}
	return nil
}

// writeJSONRPCError writes a JSON-RPC error response with the given error details.
func (s *StreamableHTTPServer) writeJSONRPCError(
	w http.ResponseWriter,
	id any,
	code int,
	message string,
) {
	response := createErrorResponse(id, code, message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		s.logger.Errorf("Failed to write JSONRPCError: %v", err)
	}
}

// --- session ---

type sessionToolsStore struct {
	mu    sync.RWMutex
	tools map[string]map[string]ServerTool // sessionID -> toolName -> tool
}

func newSessionToolsStore() *sessionToolsStore {
	return &sessionToolsStore{
		tools: make(map[string]map[string]ServerTool),
	}
}

func (s *sessionToolsStore) get(sessionID string) map[string]ServerTool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tools[sessionID]
}

func (s *sessionToolsStore) set(sessionID string, tools map[string]ServerTool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[sessionID] = tools
}

// streamableHttpSession is a session for streamable-http transport
// When in POST handlers(request/notification), it's ephemeral, and only exists in the life of the request handler.
// When in GET handlers(listening), it's a real session, and will be registered in the MCP server.
type streamableHttpSession struct {
	sessionID           string
	notificationChannel chan mcp.JSONRPCNotification // server -> client notifications
	tools               *sessionToolsStore
	authHeaders         http.Header                   // preserve authentication headers from original request
	createdAt           time.Time                     // when the session was created
	expiresAt           time.Time                     // when the session expires
}

// Default session timeout (configurable)
const DefaultSessionTimeout = 24 * time.Hour

func newStreamableHttpSession(sessionID string, toolStore *sessionToolsStore) *streamableHttpSession {
	now := time.Now()
	return &streamableHttpSession{
		sessionID:           sessionID,
		notificationChannel: make(chan mcp.JSONRPCNotification, 100),
		tools:               toolStore,
		authHeaders:         make(http.Header),
		createdAt:           now,
		expiresAt:           now.Add(DefaultSessionTimeout),
	}
}

func newStreamableHttpSessionWithHeaders(sessionID string, toolStore *sessionToolsStore, authHeaders http.Header) *streamableHttpSession {
	now := time.Now()
	return &streamableHttpSession{
		sessionID:           sessionID,
		notificationChannel: make(chan mcp.JSONRPCNotification, 100),
		tools:               toolStore,
		authHeaders:         authHeaders,
		createdAt:           now,
		expiresAt:           now.Add(DefaultSessionTimeout),
	}
}

func (s *streamableHttpSession) SessionID() string {
	return s.sessionID
}

func (s *streamableHttpSession) NotificationChannel() chan<- mcp.JSONRPCNotification {
	return s.notificationChannel
}

func (s *streamableHttpSession) Initialize() {
	// do nothing
	// the session is ephemeral, no real initialized action needed
}

func (s *streamableHttpSession) Initialized() bool {
	// the session is ephemeral, no real initialized action needed
	return true
}

var _ ClientSession = (*streamableHttpSession)(nil)

func (s *streamableHttpSession) GetSessionTools() map[string]ServerTool {
	return s.tools.get(s.sessionID)
}

func (s *streamableHttpSession) SetSessionTools(tools map[string]ServerTool) {
	s.tools.set(s.sessionID, tools)
}

func (s *streamableHttpSession) GetAuthHeaders() http.Header {
	return s.authHeaders
}

func (s *streamableHttpSession) SetAuthHeaders(headers http.Header) {
	s.authHeaders = headers
}

// SessionWithExpiration interface methods
func (s *streamableHttpSession) GetCreatedAt() time.Time {
	return s.createdAt
}

func (s *streamableHttpSession) GetExpiresAt() time.Time {
	return s.expiresAt
}

func (s *streamableHttpSession) IsExpired() bool {
	return time.Now().After(s.expiresAt)
}

func (s *streamableHttpSession) Renew(duration time.Duration) {
	s.expiresAt = time.Now().Add(duration)
}

var _ SessionWithTools = (*streamableHttpSession)(nil)
var _ SessionWithAuthHeaders = (*streamableHttpSession)(nil)
var _ SessionWithExpiration = (*streamableHttpSession)(nil)

// Session cleanup interval
const SessionCleanupInterval = 5 * time.Minute

// runSessionCleanup runs a background goroutine to clean up expired sessions
func (s *StreamableHTTPServer) runSessionCleanup() {
	defer close(s.cleanupDone)
	
	ticker := time.NewTicker(SessionCleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.cleanupCtx.Done():
			return
		case <-ticker.C:
			s.cleanupExpiredSessions()
		}
	}
}

// cleanupExpiredSessions removes expired sessions
func (s *StreamableHTTPServer) cleanupExpiredSessions() {
	var expiredSessions []string
	var totalSessions, expiringSoon int
	
	// Find expired sessions and collect health info
	s.server.sessions.Range(func(key, value any) bool {
		sessionID, ok := key.(string)
		if !ok {
			return true
		}
		
		totalSessions++
		
		if sessionWithExp, ok := value.(SessionWithExpiration); ok {
			if sessionWithExp.IsExpired() {
				expiredSessions = append(expiredSessions, sessionID)
			} else {
				// Check if session expires within 30 minutes
				if time.Until(sessionWithExp.GetExpiresAt()) < 30*time.Minute {
					expiringSoon++
				}
			}
		}
		return true
	})
	
	// Remove expired sessions
	for _, sessionID := range expiredSessions {
		s.logger.Infof("Cleaning up expired session: %s", sessionID)
		s.server.UnregisterSession(context.Background(), sessionID)
	}
	
	// Log session health status
	activeSessions := totalSessions - len(expiredSessions)
	if len(expiredSessions) > 0 || expiringSoon > 0 {
		s.logger.Infof("Session health: %d active, %d expired (cleaned), %d expiring soon", 
			activeSessions, len(expiredSessions), expiringSoon)
	}
}

// GetSessionHealth returns current session health statistics
func (s *StreamableHTTPServer) GetSessionHealth() (total, active, expiringSoon, expired int) {
	s.server.sessions.Range(func(key, value any) bool {
		_, ok := key.(string)
		if !ok {
			return true
		}
		
		total++
		
		if sessionWithExp, ok := value.(SessionWithExpiration); ok {
			if sessionWithExp.IsExpired() {
				expired++
			} else {
				active++
				// Check if session expires within 30 minutes
				if time.Until(sessionWithExp.GetExpiresAt()) < 30*time.Minute {
					expiringSoon++
				}
			}
		} else {
			// Non-expiring sessions are considered active
			active++
		}
		return true
	})
	
	return total, active, expiringSoon, expired
}

// --- session id manager ---

type SessionIdManager interface {
	Generate() string
	// Validate checks if a session ID is valid and not terminated.
	// Returns isTerminated=true if the ID is valid but belongs to a terminated session.
	// Returns err!=nil if the ID format is invalid or lookup failed.
	Validate(sessionID string) (isTerminated bool, err error)
	// Terminate marks a session ID as terminated.
	// Returns isNotAllowed=true if the server policy prevents client termination.
	// Returns err!=nil if the ID is invalid or termination failed.
	Terminate(sessionID string) (isNotAllowed bool, err error)
}

// StatelessSessionIdManager does nothing, which means it has no session management, which is stateless.
type StatelessSessionIdManager struct{}

func (s *StatelessSessionIdManager) Generate() string {
	return ""
}

func (s *StatelessSessionIdManager) Validate(sessionID string) (isTerminated bool, err error) {
	if sessionID != "" {
		return false, fmt.Errorf("session id is not allowed to be set when stateless")
	}
	return false, nil
}

func (s *StatelessSessionIdManager) Terminate(sessionID string) (isNotAllowed bool, err error) {
	return false, nil
}

// InsecureStatefulSessionIdManager generate id with uuid
// It won't validate the id indeed, so it could be fake.
// For more secure session id, use a more complex generator, like a JWT.
type InsecureStatefulSessionIdManager struct{}

const idPrefix = "mcp-session-"

func (s *InsecureStatefulSessionIdManager) Generate() string {
	return idPrefix + uuid.New().String()
}

func (s *InsecureStatefulSessionIdManager) Validate(sessionID string) (isTerminated bool, err error) {
	// validate the session id is a valid uuid
	if !strings.HasPrefix(sessionID, idPrefix) {
		return false, fmt.Errorf("invalid session id: %s", sessionID)
	}
	if _, err := uuid.Parse(sessionID[len(idPrefix):]); err != nil {
		return false, fmt.Errorf("invalid session id: %s", sessionID)
	}
	return false, nil
}

func (s *InsecureStatefulSessionIdManager) Terminate(sessionID string) (isNotAllowed bool, err error) {
	return false, nil
}

// NewTestStreamableHTTPServer creates a test server for testing purposes
func NewTestStreamableHTTPServer(server *MCPServer, opts ...StreamableHTTPOption) *httptest.Server {
	sseServer := NewStreamableHTTPServer(server, opts...)
	testServer := httptest.NewServer(sseServer)
	return testServer
}

// handleToolsAPI provides an optimized HTTP API for listing tools with compression and caching
func (s *StreamableHTTPServer) handleToolsAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Apply HTTP context function if available
	if s.contextFunc != nil {
		ctx = s.contextFunc(ctx, r)
	}
	
	// Create a temporary session for tool listing
	sessionID := uuid.New().String()
	session := newStreamableHttpSession(sessionID, s.sessionTools)
	
	if err := s.server.RegisterSession(ctx, session); err != nil {
		http.Error(w, fmt.Sprintf("Session registration failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer s.server.UnregisterSession(ctx, sessionID)
	
	// Get tools using MCP protocol
	toolsRequest := mcp.ListToolsRequest{}
	
	result, reqErr := s.server.handleListTools(ctx, "tools-api", toolsRequest)
	if reqErr != nil {
		http.Error(w, fmt.Sprintf("Failed to list tools: %v", reqErr.err), http.StatusInternalServerError)
		return
	}
	
	// Check query parameters for optimization options
	query := r.URL.Query()
	// Enable compression by default, allow explicit override
	compressedParam := query.Get("compressed")
	compressed := compressedParam == "" || compressedParam == "true" || strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
	// Use compact mode by default for tools endpoint, allow explicit override
	compactParam := query.Get("compact")
	compact := compactParam == "" || compactParam == "true"
	limit := 0
	if limitStr := query.Get("limit"); limitStr != "" {
		if parsedLimit, err := json.Number(limitStr).Int64(); err == nil && parsedLimit > 0 {
			limit = int(parsedLimit)
		}
	}
	
	// Optimize response based on parameters
	tools := result.Tools
	if limit > 0 && len(tools) > limit {
		tools = tools[:limit]
		// Add pagination info
		w.Header().Set("X-Total-Tools", fmt.Sprintf("%d", len(result.Tools)))
		w.Header().Set("X-Returned-Tools", fmt.Sprintf("%d", limit))
	}
	
	// Set appropriate headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300") // 5 minute cache
	
	var responseData []byte
	var err error
	
	if compact {
		// Return compact format with just name and description
		compactTools := make([]map[string]any, len(tools))
		for i, tool := range tools {
			// Sanitize description to ensure valid JSON
			sanitizedDesc := strings.ReplaceAll(tool.Description, "\x00", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x01", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x02", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x03", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x04", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x05", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x06", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x07", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x08", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x0b", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x0c", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x0e", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x0f", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x10", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x11", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x12", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x13", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x14", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x15", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x16", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x17", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x18", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x19", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x1a", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x1b", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x1c", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x1d", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x1e", "")
			sanitizedDesc = strings.ReplaceAll(sanitizedDesc, "\x1f", "")
			
			compactTools[i] = map[string]any{
				"name":        tool.Name,
				"description": sanitizedDesc,
			}
		}
		responseData, err = json.Marshal(compactTools)
	} else {
		responseData, err = json.Marshal(tools)
	}
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to serialize tools: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Apply compression if supported
	if compressed && len(responseData) > 1024 { // Only compress if > 1KB
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		
		gz := gzip.NewWriter(w)
		defer gz.Close()
		
		w.WriteHeader(http.StatusOK)
		_, err = gz.Write(responseData)
		if err != nil {
			// Log error but don't return error to client as headers are already sent
			fmt.Printf("Compression error: %v\n", err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(responseData)
		if err != nil {
			fmt.Printf("Write error: %v\n", err)
		}
	}
}

// logIncomingRequest logs detailed information about incoming HTTP requests
func (s *StreamableHTTPServer) logIncomingRequest(r *http.Request) {
	timestamp := time.Now().Format("2006-01-02 15:04:05 MST")
	
	log.Printf("┌─ INCOMING MCP REQUEST ────────────────────────────────────────────────────────")
	log.Printf("│ 🕐 %s", timestamp)
	log.Printf("│ 🌐 %s %s", r.Method, r.URL.String())
	log.Printf("│ 📍 Remote: %s", r.RemoteAddr)
	
	// Log all headers
	if len(r.Header) > 0 {
		log.Printf("│ 📋 Headers:")
		for name, values := range r.Header {
			// Show auth headers but mask sensitive values
			if strings.Contains(strings.ToLower(name), "auth") || 
			   strings.Contains(strings.ToLower(name), "key") ||
			   strings.Contains(strings.ToLower(name), "token") {
				log.Printf("│    %s: %s", name, maskSensitiveValue(strings.Join(values, ", ")))
			} else {
				log.Printf("│    %s: %s", name, strings.Join(values, ", "))
			}
		}
	}
	
	// Log query parameters
	if len(r.URL.RawQuery) > 0 {
		log.Printf("│ 🔍 Query: %s", r.URL.RawQuery)
	}
	
	// Log request body for POST requests (with size limit)
	if r.Method == "POST" && r.ContentLength > 0 && r.ContentLength < 10240 { // Max 10KB
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil {
			// Restore body for actual processing
			r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
			
			bodyStr := string(bodyBytes)
			if len(bodyStr) > 2000 {
				bodyStr = bodyStr[:2000] + "... [truncated]"
			}
			log.Printf("│ 📦 Body: %s", bodyStr)
		}
	}
	
	log.Printf("└───────────────────────────────────────────────────────────────────────────────")
}

// maskSensitiveValue masks sensitive authentication values for logging
func maskSensitiveValue(value string) string {
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	// Show first 4 and last 4 characters
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}
