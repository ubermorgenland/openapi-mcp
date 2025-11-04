package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// MemoryEfficientSpecLoader loads and processes OpenAPI specifications with memory optimization
type MemoryEfficientSpecLoader struct {
	processor     *StreamProcessor
	maxSpecSizeMB int64
}

// NewMemoryEfficientSpecLoader creates a new memory-efficient spec loader
func NewMemoryEfficientSpecLoader(maxMemoryMB, maxSpecSizeMB int64) *MemoryEfficientSpecLoader {
	return &MemoryEfficientSpecLoader{
		processor:     NewStreamProcessor(maxMemoryMB, 32768), // 32KB chunks for specs
		maxSpecSizeMB: maxSpecSizeMB,
	}
}

// LoadSpecStreaming loads an OpenAPI spec from a reader with memory management
func (mesl *MemoryEfficientSpecLoader) LoadSpecStreaming(ctx context.Context, reader io.Reader) (*openapi3.T, error) {
	// Use buffered reading to control memory usage
	buffer := mesl.processor.GetBuffer()
	defer mesl.processor.PutBuffer(buffer)
	
	// Read spec content in chunks
	chunk := mesl.processor.GetByteSlice()
	defer mesl.processor.PutByteSlice(chunk)
	
	var totalSize int64
	
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		// Check memory usage
		if !mesl.processor.CheckMemory() {
			return nil, fmt.Errorf("memory usage exceeded limits while loading spec")
		}
		
		n, err := reader.Read(chunk[:cap(chunk)])
		if n > 0 {
			totalSize += int64(n)
			
			// Check spec size limit
			if totalSize > mesl.maxSpecSizeMB*1024*1024 {
				return nil, fmt.Errorf("spec size (%dMB) exceeds maximum allowed size (%dMB)", 
					totalSize/(1024*1024), mesl.maxSpecSizeMB)
			}
			
			buffer.Write(chunk[:n])
		}
		
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error reading spec: %w", err)
		}
	}
	
	log.Printf("Loaded spec content: %dMB", totalSize/(1024*1024))
	
	// Parse the spec
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("error parsing OpenAPI spec: %w", err)
	}
	
	// Validate the spec
	if err := doc.Validate(ctx); err != nil {
		return nil, fmt.Errorf("spec validation failed: %w", err)
	}
	
	return doc, nil
}

// OptimizeSpec optimizes a spec by removing unnecessary fields to reduce memory usage
func (mesl *MemoryEfficientSpecLoader) OptimizeSpec(spec *openapi3.T) error {
	if spec == nil {
		return fmt.Errorf("spec cannot be nil")
	}
	
	// Remove examples from schema to save memory
	if spec.Components != nil && spec.Components.Schemas != nil {
		for _, schemaRef := range spec.Components.Schemas {
			if schemaRef.Value != nil {
				mesl.optimizeSchema(schemaRef.Value)
			}
		}
	}
	
	// Optimize paths
	if spec.Paths != nil {
		for _, pathItem := range spec.Paths {
			if pathItem != nil {
				mesl.optimizePathItem(pathItem)
			}
		}
	}
	
	log.Printf("Optimized spec for %s v%s", spec.Info.Title, spec.Info.Version)
	return nil
}

// optimizeSchema removes memory-intensive fields from schemas
func (mesl *MemoryEfficientSpecLoader) optimizeSchema(schema *openapi3.Schema) {
	if schema == nil {
		return
	}
	
	// Remove examples to save memory
	schema.Example = nil
	
	// Recursively optimize nested schemas
	if schema.Properties != nil {
		for _, propRef := range schema.Properties {
			if propRef.Value != nil {
				mesl.optimizeSchema(propRef.Value)
			}
		}
	}
	
	if schema.Items != nil && schema.Items.Value != nil {
		mesl.optimizeSchema(schema.Items.Value)
	}
	
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil && schema.AdditionalProperties.Schema.Value != nil {
		mesl.optimizeSchema(schema.AdditionalProperties.Schema.Value)
	}
}

// optimizePathItem removes memory-intensive fields from path items
func (mesl *MemoryEfficientSpecLoader) optimizePathItem(pathItem *openapi3.PathItem) {
	if pathItem == nil {
		return
	}
	
	operations := []*openapi3.Operation{
		pathItem.Get, pathItem.Post, pathItem.Put, pathItem.Delete,
		pathItem.Options, pathItem.Head, pathItem.Patch, pathItem.Trace,
	}
	
	for _, op := range operations {
		if op != nil {
			mesl.optimizeOperation(op)
		}
	}
}

// optimizeOperation removes memory-intensive fields from operations
func (mesl *MemoryEfficientSpecLoader) optimizeOperation(op *openapi3.Operation) {
	if op == nil {
		return
	}
	
	// Keep description but remove lengthy examples
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		for _, contentType := range op.RequestBody.Value.Content {
			if contentType.Examples != nil {
				contentType.Examples = nil // Remove examples
			}
		}
	}
	
	if op.Responses != nil {
		for _, responseRef := range op.Responses {
			if responseRef.Value != nil && responseRef.Value.Content != nil {
				for _, contentType := range responseRef.Value.Content {
					if contentType.Examples != nil {
						contentType.Examples = nil // Remove examples
					}
				}
			}
		}
	}
}

// SpecSummary provides a lightweight summary of a spec for memory-efficient storage
type SpecSummary struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	PathCount   int    `json:"path_count"`
	MethodCount int    `json:"method_count"`
	SizeBytes   int64  `json:"size_bytes"`
}

// CreateSpecSummary creates a lightweight summary of an OpenAPI spec
func (mesl *MemoryEfficientSpecLoader) CreateSpecSummary(spec *openapi3.T, originalSize int64) *SpecSummary {
	summary := &SpecSummary{
		SizeBytes: originalSize,
	}
	
	if spec.Info != nil {
		summary.Title = spec.Info.Title
		summary.Version = spec.Info.Version
	}
	
	if spec.Paths != nil {
		summary.PathCount = len(spec.Paths)
		
		// Count methods across all paths
		for _, pathItem := range spec.Paths {
			if pathItem != nil {
				if pathItem.Get != nil {
					summary.MethodCount++
				}
				if pathItem.Post != nil {
					summary.MethodCount++
				}
				if pathItem.Put != nil {
					summary.MethodCount++
				}
				if pathItem.Delete != nil {
					summary.MethodCount++
				}
				if pathItem.Options != nil {
					summary.MethodCount++
				}
				if pathItem.Head != nil {
					summary.MethodCount++
				}
				if pathItem.Patch != nil {
					summary.MethodCount++
				}
				if pathItem.Trace != nil {
					summary.MethodCount++
				}
			}
		}
	}
	
	return summary
}

// GetMemoryStats returns current memory usage statistics
func (mesl *MemoryEfficientSpecLoader) GetMemoryStats() (allocMB, sysMB int64) {
	return mesl.processor.GetMemoryStats()
}

// EstimateSpecMemoryUsage estimates the memory usage of a spec
func EstimateSpecMemoryUsage(spec *openapi3.T) int64 {
	if spec == nil {
		return 0
	}
	
	// Convert to JSON to estimate serialized size
	data, err := json.Marshal(spec)
	if err != nil {
		return 0
	}
	
	// Estimate memory usage as roughly 3x the JSON size
	// (due to Go's internal representation and additional metadata)
	return int64(len(data)) * 3
}

// CompressSpecForStorage compresses spec data for storage
func CompressSpecForStorage(spec *openapi3.T) ([]byte, error) {
	if spec == nil {
		return nil, fmt.Errorf("spec cannot be nil")
	}
	
	// First optimize the spec
	optimizedSpec := *spec // Shallow copy
	
	// Create a minimal version for storage
	minimalSpec := &openapi3.T{
		OpenAPI: spec.OpenAPI,
		Info:    spec.Info,
		Paths:   spec.Paths,
		Components: &openapi3.Components{
			Schemas:         spec.Components.Schemas,
			SecuritySchemes: spec.Components.SecuritySchemes,
		},
	}
	
	// Marshal to JSON
	data, err := json.Marshal(minimalSpec)
	if err != nil {
		return nil, fmt.Errorf("error marshaling spec: %w", err)
	}
	
	return data, nil
}