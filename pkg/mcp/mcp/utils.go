package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cast"
)

// ClientRequest types
var (
	_ ClientRequest = &PingRequest{}
	_ ClientRequest = &InitializeRequest{}
	_ ClientRequest = &CompleteRequest{}
	_ ClientRequest = &SetLevelRequest{}
	_ ClientRequest = &GetPromptRequest{}
	_ ClientRequest = &ListPromptsRequest{}
	_ ClientRequest = &ListResourcesRequest{}
	_ ClientRequest = &ReadResourceRequest{}
	_ ClientRequest = &SubscribeRequest{}
	_ ClientRequest = &UnsubscribeRequest{}
	_ ClientRequest = &CallToolRequest{}
	_ ClientRequest = &ListToolsRequest{}
)

// ClientNotification types
var (
	_ ClientNotification = &CancelledNotification{}
	_ ClientNotification = &ProgressNotification{}
	_ ClientNotification = &InitializedNotification{}
	_ ClientNotification = &RootsListChangedNotification{}
)

// ClientResult types
var (
	_ ClientResult = &EmptyResult{}
	_ ClientResult = &CreateMessageResult{}
	_ ClientResult = &ListRootsResult{}
)

// ServerRequest types
var (
	_ ServerRequest = &PingRequest{}
	_ ServerRequest = &CreateMessageRequest{}
	_ ServerRequest = &ListRootsRequest{}
)

// ServerNotification types
var (
	_ ServerNotification = &CancelledNotification{}
	_ ServerNotification = &ProgressNotification{}
	_ ServerNotification = &LoggingMessageNotification{}
	_ ServerNotification = &ResourceUpdatedNotification{}
	_ ServerNotification = &ResourceListChangedNotification{}
	_ ServerNotification = &ToolListChangedNotification{}
	_ ServerNotification = &PromptListChangedNotification{}
)

// ServerResult types
var (
	_ ServerResult = &EmptyResult{}
	_ ServerResult = &InitializeResult{}
	_ ServerResult = &CompleteResult{}
	_ ServerResult = &GetPromptResult{}
	_ ServerResult = &ListPromptsResult{}
	_ ServerResult = &ListResourcesResult{}
	_ ServerResult = &ReadResourceResult{}
	_ ServerResult = &CallToolResult{}
	_ ServerResult = &ListToolsResult{}
)

// Helper functions for type assertions

// asType attempts to cast the given interface to the given type
func asType[T any](content any) (*T, bool) {
	tc, ok := content.(T)
	if !ok {
		return nil, false
	}
	return &tc, true
}

// AsTextContent attempts to cast the given interface to TextContent
func AsTextContent(content any) (*TextContent, bool) {
	return asType[TextContent](content)
}

// AsImageContent attempts to cast the given interface to ImageContent
func AsImageContent(content any) (*ImageContent, bool) {
	return asType[ImageContent](content)
}

// AsAudioContent attempts to cast the given interface to AudioContent
func AsAudioContent(content any) (*AudioContent, bool) {
	return asType[AudioContent](content)
}

// AsEmbeddedResource attempts to cast the given interface to EmbeddedResource
func AsEmbeddedResource(content any) (*EmbeddedResource, bool) {
	return asType[EmbeddedResource](content)
}

// AsTextResourceContents attempts to cast the given interface to TextResourceContents
func AsTextResourceContents(content any) (*TextResourceContents, bool) {
	return asType[TextResourceContents](content)
}

// AsBlobResourceContents attempts to cast the given interface to BlobResourceContents
func AsBlobResourceContents(content any) (*BlobResourceContents, bool) {
	return asType[BlobResourceContents](content)
}

// Helper function for JSON-RPC

// NewJSONRPCResponse creates a new JSONRPCResponse with the given id and result
func NewJSONRPCResponse(id RequestId, result Result, resultType string) JSONRPCResponse {
	result.ResultType = resultType
	return JSONRPCResponse{
		JSONRPC: JSONRPC_VERSION,
		ID:      id,
		Result:  result,
	}
}

// NewJSONRPCError creates a new JSONRPCResponse with the given id, code, and message
func NewJSONRPCError(
	id RequestId,
	code int,
	message string,
	details any,
) JSONRPCError {
	return JSONRPCError{
		JSONRPC: JSONRPC_VERSION,
		ID:      id,
		Error: struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Details any    `json:"details,omitempty"`
		}{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// NewProgressNotification
// Helper function for creating a progress notification
func NewProgressNotification(
	token ProgressToken,
	progress float64,
	total *float64,
	message *string,
) ProgressNotification {
	notification := ProgressNotification{
		Notification: Notification{
			Method: "notifications/progress",
		},
		Params: struct {
			ProgressToken ProgressToken `json:"progressToken"`
			Progress      float64       `json:"progress"`
			Total         float64       `json:"total,omitempty"`
			Message       string        `json:"message,omitempty"`
		}{
			ProgressToken: token,
			Progress:      progress,
		},
	}
	if total != nil {
		notification.Params.Total = *total
	}
	if message != nil {
		notification.Params.Message = *message
	}
	return notification
}

// NewLoggingMessageNotification
// Helper function for creating a logging message notification
func NewLoggingMessageNotification(
	level LoggingLevel,
	logger string,
	data any,
) LoggingMessageNotification {
	return LoggingMessageNotification{
		Notification: Notification{
			Method: "notifications/message",
		},
		Params: struct {
			Level  LoggingLevel `json:"level"`
			Logger string       `json:"logger,omitempty"`
			Data   any          `json:"data"`
		}{
			Level:  level,
			Logger: logger,
			Data:   data,
		},
	}
}

// NewPromptMessage
// Helper function to create a new PromptMessage
func NewPromptMessage(role Role, content Content) PromptMessage {
	return PromptMessage{
		Role:    role,
		Content: content,
	}
}

// NewTextContent
// Helper function to create a new TextContent
func NewTextContent(text string) TextContent {
	return TextContent{
		Type: "text",
		Text: text,
	}
}

// NewImageContent
// Helper function to create a new ImageContent
func NewImageContent(data, mimeType string) ImageContent {
	return ImageContent{
		Type:     "image",
		Data:     data,
		MIMEType: mimeType,
	}
}

// Helper function to create a new AudioContent
func NewAudioContent(data, mimeType string) AudioContent {
	return AudioContent{
		Type:     "audio",
		Data:     data,
		MIMEType: mimeType,
	}
}

// Helper function to create a new EmbeddedResource
func NewEmbeddedResource(resource ResourceContents) EmbeddedResource {
	return EmbeddedResource{
		Type:     "resource",
		Resource: resource,
	}
}

// NewToolResultText creates a new CallToolResult with a text content
func NewToolResultText(text string, schema map[string]any, arguments map[string]any, examples []any, usage string, nextSteps []string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: text,
			},
		},
		Schema:    schema,
		Arguments: arguments,
		Examples:  examples,
		Usage:     usage,
		NextSteps: nextSteps,
	}
}

// NewToolResultImage creates a new CallToolResult with both text and image content
func NewToolResultImage(text, imageData, mimeType string, schema map[string]any, arguments map[string]any, examples []any, usage string, nextSteps []string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: text,
			},
			ImageContent{
				Type:     "image",
				Data:     imageData,
				MIMEType: mimeType,
			},
		},
		Schema:    schema,
		Arguments: arguments,
		Examples:  examples,
		Usage:     usage,
		NextSteps: nextSteps,
	}
}

// NewToolResultAudio creates a new CallToolResult with both text and audio content
func NewToolResultAudio(text, imageData, mimeType string, schema map[string]any, arguments map[string]any, examples []any, usage string, nextSteps []string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: text,
			},
			AudioContent{
				Type:     "audio",
				Data:     imageData,
				MIMEType: mimeType,
			},
		},
		Schema:    schema,
		Arguments: arguments,
		Examples:  examples,
		Usage:     usage,
		NextSteps: nextSteps,
	}
}

// NewToolResultResource creates a new CallToolResult with an embedded resource
func NewToolResultResource(text string, resource ResourceContents, schema map[string]any, arguments map[string]any, examples []any, usage string, nextSteps []string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: text,
			},
			EmbeddedResource{
				Type:     "resource",
				Resource: resource,
			},
		},
		Schema:    schema,
		Arguments: arguments,
		Examples:  examples,
		Usage:     usage,
		NextSteps: nextSteps,
	}
}

// NewToolResultError creates a new CallToolResult with an error message.
// Any errors that originate from the tool SHOULD be reported inside the result object.
func NewToolResultError(text string, schema map[string]any, arguments map[string]any, examples []any, usage string, nextSteps []string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: text,
			},
		},
		IsError:   true,
		Schema:    schema,
		Arguments: arguments,
		Examples:  examples,
		Usage:     usage,
		NextSteps: nextSteps,
	}
}

// NewToolResultErrorFromErr creates a new CallToolResult with an error message.
// If an error is provided, its details will be appended to the text message.
// Any errors that originate from the tool SHOULD be reported inside the result object.
func NewToolResultErrorFromErr(text string, err error, schema map[string]any, arguments map[string]any, examples []any, usage string, nextSteps []string) *CallToolResult {
	if err != nil {
		text = fmt.Sprintf("%s: %v", text, err)
	}
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: text,
			},
		},
		IsError:   true,
		Schema:    schema,
		Arguments: arguments,
		Examples:  examples,
		Usage:     usage,
		NextSteps: nextSteps,
	}
}

// Helper for formatting numbers in tool results
func FormatNumberResult(value float64) *CallToolResult {
	return NewToolResultText(fmt.Sprintf("%.2f", value), nil, nil, nil, "", nil)
}

func ExtractString(data map[string]any, key string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func ExtractMap(data map[string]any, key string) map[string]any {
	if value, ok := data[key]; ok {
		if m, ok := value.(map[string]any); ok {
			return m
		}
	}
	return nil
}

func ParseContent(contentMap map[string]any) (Content, error) {
	contentType := ExtractString(contentMap, "type")

	switch contentType {
	case "text":
		text := ExtractString(contentMap, "text")
		return NewTextContent(text), nil

	case "image":
		data := ExtractString(contentMap, "data")
		mimeType := ExtractString(contentMap, "mimeType")
		if data == "" || mimeType == "" {
			return nil, fmt.Errorf("image data or mimeType is missing")
		}
		return NewImageContent(data, mimeType), nil

	case "audio":
		data := ExtractString(contentMap, "data")
		mimeType := ExtractString(contentMap, "mimeType")
		if data == "" || mimeType == "" {
			return nil, fmt.Errorf("audio data or mimeType is missing")
		}
		return NewAudioContent(data, mimeType), nil

	case "resource":
		resourceMap := ExtractMap(contentMap, "resource")
		if resourceMap == nil {
			return nil, fmt.Errorf("resource is missing")
		}

		resourceContents, err := ParseResourceContents(resourceMap)
		if err != nil {
			return nil, err
		}

		return NewEmbeddedResource(resourceContents), nil
	}

	return nil, fmt.Errorf("unsupported content type: %s", contentType)
}

func ParseGetPromptResult(rawMessage *json.RawMessage) (*GetPromptResult, error) {
	if rawMessage == nil {
		return nil, fmt.Errorf("response is nil")
	}

	var jsonContent map[string]any
	if err := json.Unmarshal(*rawMessage, &jsonContent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	result := GetPromptResult{}

	meta, ok := jsonContent["_meta"]
	if ok {
		if metaMap, ok := meta.(map[string]any); ok {
			result.Meta = metaMap
		}
	}

	description, ok := jsonContent["description"]
	if ok {
		if descriptionStr, ok := description.(string); ok {
			result.Description = descriptionStr
		}
	}

	messages, ok := jsonContent["messages"]
	if ok {
		messagesArr, ok := messages.([]any)
		if !ok {
			return nil, fmt.Errorf("messages is not an array")
		}

		for _, message := range messagesArr {
			messageMap, ok := message.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("message is not an object")
			}

			// Extract role
			roleStr := ExtractString(messageMap, "role")
			if roleStr == "" || (roleStr != string(RoleAssistant) && roleStr != string(RoleUser)) {
				return nil, fmt.Errorf("unsupported role: %s", roleStr)
			}

			// Extract content
			contentMap, ok := messageMap["content"].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("content is not an object")
			}

			// Process content
			content, err := ParseContent(contentMap)
			if err != nil {
				return nil, err
			}

			// Append processed message
			result.Messages = append(result.Messages, NewPromptMessage(Role(roleStr), content))

		}
	}

	return &result, nil
}

func ParseCallToolResult(rawMessage *json.RawMessage) (*CallToolResult, error) {
	if rawMessage == nil {
		return nil, fmt.Errorf("response is nil")
	}

	var jsonContent map[string]any
	if err := json.Unmarshal(*rawMessage, &jsonContent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var result CallToolResult

	meta, ok := jsonContent["_meta"]
	if ok {
		if metaMap, ok := meta.(map[string]any); ok {
			result.Meta = metaMap
		}
	}

	isError, ok := jsonContent["isError"]
	if ok {
		if isErrorBool, ok := isError.(bool); ok {
			result.IsError = isErrorBool
		}
	}

	contents, ok := jsonContent["content"]
	if !ok {
		return nil, fmt.Errorf("content is missing")
	}

	contentArr, ok := contents.([]any)
	if !ok {
		return nil, fmt.Errorf("content is not an array")
	}

	for _, content := range contentArr {
		// Extract content
		contentMap, ok := content.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("content is not an object")
		}

		// Process content
		content, err := ParseContent(contentMap)
		if err != nil {
			return nil, err
		}

		result.Content = append(result.Content, content)
	}

	return &result, nil
}

func ParseResourceContents(contentMap map[string]any) (ResourceContents, error) {
	uri := ExtractString(contentMap, "uri")
	if uri == "" {
		return nil, fmt.Errorf("resource uri is missing")
	}

	mimeType := ExtractString(contentMap, "mimeType")

	if text := ExtractString(contentMap, "text"); text != "" {
		return TextResourceContents{
			URI:      uri,
			MIMEType: mimeType,
			Text:     text,
		}, nil
	}

	if blob := ExtractString(contentMap, "blob"); blob != "" {
		return BlobResourceContents{
			URI:      uri,
			MIMEType: mimeType,
			Blob:     blob,
		}, nil
	}

	return nil, fmt.Errorf("unsupported resource type")
}

func ParseReadResourceResult(rawMessage *json.RawMessage) (*ReadResourceResult, error) {
	if rawMessage == nil {
		return nil, fmt.Errorf("response is nil")
	}

	var jsonContent map[string]any
	if err := json.Unmarshal(*rawMessage, &jsonContent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var result ReadResourceResult

	meta, ok := jsonContent["_meta"]
	if ok {
		if metaMap, ok := meta.(map[string]any); ok {
			result.Meta = metaMap
		}
	}

	contents, ok := jsonContent["contents"]
	if !ok {
		return nil, fmt.Errorf("contents is missing")
	}

	contentArr, ok := contents.([]any)
	if !ok {
		return nil, fmt.Errorf("contents is not an array")
	}

	for _, content := range contentArr {
		// Extract content
		contentMap, ok := content.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("content is not an object")
		}

		// Process content
		content, err := ParseResourceContents(contentMap)
		if err != nil {
			return nil, err
		}

		result.Contents = append(result.Contents, content)
	}

	return &result, nil
}

func ParseArgument(request CallToolRequest, key string, defaultVal any) any {
	args := request.GetArguments()
	if _, ok := args[key]; !ok {
		return defaultVal
	} else {
		return args[key]
	}
}

// ParseBoolean extracts and converts a boolean parameter from a CallToolRequest.
// If the key is not found in the Arguments map, the defaultValue is returned.
// The function uses cast.ToBool for conversion which handles various string representations
// such as "true", "yes", "1", etc.
func ParseBoolean(request CallToolRequest, key string, defaultValue bool) bool {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToBool(v)
}

// ParseInt64 extracts and converts an int64 parameter from a CallToolRequest.
// If the key is not found in the Arguments map, the defaultValue is returned.
func ParseInt64(request CallToolRequest, key string, defaultValue int64) int64 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToInt64(v)
}

// ParseInt32 extracts and converts an int32 parameter from a CallToolRequest.
func ParseInt32(request CallToolRequest, key string, defaultValue int32) int32 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToInt32(v)
}

// ParseInt16 extracts and converts an int16 parameter from a CallToolRequest.
func ParseInt16(request CallToolRequest, key string, defaultValue int16) int16 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToInt16(v)
}

// ParseInt8 extracts and converts an int8 parameter from a CallToolRequest.
func ParseInt8(request CallToolRequest, key string, defaultValue int8) int8 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToInt8(v)
}

// ParseInt extracts and converts an int parameter from a CallToolRequest.
func ParseInt(request CallToolRequest, key string, defaultValue int) int {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToInt(v)
}

// ParseUInt extracts and converts an uint parameter from a CallToolRequest.
func ParseUInt(request CallToolRequest, key string, defaultValue uint) uint {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToUint(v)
}

// ParseUInt64 extracts and converts an uint64 parameter from a CallToolRequest.
func ParseUInt64(request CallToolRequest, key string, defaultValue uint64) uint64 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToUint64(v)
}

// ParseUInt32 extracts and converts an uint32 parameter from a CallToolRequest.
func ParseUInt32(request CallToolRequest, key string, defaultValue uint32) uint32 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToUint32(v)
}

// ParseUInt16 extracts and converts an uint16 parameter from a CallToolRequest.
func ParseUInt16(request CallToolRequest, key string, defaultValue uint16) uint16 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToUint16(v)
}

// ParseUInt8 extracts and converts an uint8 parameter from a CallToolRequest.
func ParseUInt8(request CallToolRequest, key string, defaultValue uint8) uint8 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToUint8(v)
}

// ParseFloat32 extracts and converts a float32 parameter from a CallToolRequest.
func ParseFloat32(request CallToolRequest, key string, defaultValue float32) float32 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToFloat32(v)
}

// ParseFloat64 extracts and converts a float64 parameter from a CallToolRequest.
func ParseFloat64(request CallToolRequest, key string, defaultValue float64) float64 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToFloat64(v)
}

// ParseString extracts and converts a string parameter from a CallToolRequest.
func ParseString(request CallToolRequest, key string, defaultValue string) string {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToString(v)
}

// ParseStringMap extracts and converts a string map parameter from a CallToolRequest.
func ParseStringMap(request CallToolRequest, key string, defaultValue map[string]any) map[string]any {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToStringMap(v)
}

// ToBoolPtr returns a pointer to the given boolean value
func ToBoolPtr(b bool) *bool {
	return &b
}
