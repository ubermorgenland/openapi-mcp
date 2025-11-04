package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
)

// StreamingJSONProcessor handles large JSON responses with memory-efficient streaming
type StreamingJSONProcessor struct {
	processor *StreamProcessor
}

// NewStreamingJSONProcessor creates a new streaming JSON processor
func NewStreamingJSONProcessor(maxMemoryMB int64) *StreamingJSONProcessor {
	return &StreamingJSONProcessor{
		processor: NewStreamProcessor(maxMemoryMB, 8192),
	}
}

// ProcessLargeJSON processes large JSON data in chunks to avoid memory issues
func (sjp *StreamingJSONProcessor) ProcessLargeJSON(ctx context.Context, reader io.Reader, callback func(interface{}) error) error {
	decoder := json.NewDecoder(reader)
	
	// Configure decoder for large numbers
	decoder.UseNumber()
	
	var processedCount int
	
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Check memory usage periodically
		if processedCount%100 == 0 && !sjp.processor.CheckMemory() {
			return fmt.Errorf("memory usage exceeded limits during JSON processing")
		}
		
		var item interface{}
		if err := decoder.Decode(&item); err != nil {
			if err == io.EOF {
				break // End of stream
			}
			return fmt.Errorf("JSON decode error: %w", err)
		}
		
		// Process the item
		if err := callback(item); err != nil {
			return fmt.Errorf("callback error: %w", err)
		}
		
		processedCount++
		
		// Log progress for large datasets
		if processedCount%1000 == 0 {
			allocMB, sysMB := sjp.processor.GetMemoryStats()
			log.Printf("Processed %d items, Memory: %dMB alloc, %dMB sys", processedCount, allocMB, sysMB)
		}
	}
	
	log.Printf("Successfully processed %d JSON items", processedCount)
	return nil
}

// ChunkedResponseWriter writes responses in chunks to manage memory usage
type ChunkedResponseWriter struct {
	writer    io.Writer
	processor *StreamProcessor
	buffer    []byte
	written   int64
}

// NewChunkedResponseWriter creates a new chunked response writer
func NewChunkedResponseWriter(writer io.Writer, maxMemoryMB int64) *ChunkedResponseWriter {
	processor := NewStreamProcessor(maxMemoryMB, 8192)
	return &ChunkedResponseWriter{
		writer:    writer,
		processor: processor,
		buffer:    processor.GetByteSlice(),
	}
}

// Write implements io.Writer interface with memory management
func (crw *ChunkedResponseWriter) Write(p []byte) (n int, err error) {
	// Check memory usage before large writes
	if len(p) > 1024*1024 && !crw.processor.CheckMemory() { // 1MB threshold
		return 0, fmt.Errorf("memory usage exceeded limits during write")
	}
	
	// For large writes, stream directly without buffering
	if len(p) > crw.processor.GetChunkSize() {
		written, err := crw.writer.Write(p)
		crw.written += int64(written)
		return written, err
	}
	
	// Buffer small writes for efficiency
	if len(crw.buffer)+len(p) > cap(crw.buffer) {
		// Flush buffer first
		if err := crw.flush(); err != nil {
			return 0, err
		}
	}
	
	// Add to buffer
	crw.buffer = append(crw.buffer, p...)
	return len(p), nil
}

// flush writes the buffer content to the underlying writer
func (crw *ChunkedResponseWriter) flush() error {
	if len(crw.buffer) == 0 {
		return nil
	}
	
	written, err := crw.writer.Write(crw.buffer)
	crw.written += int64(written)
	crw.buffer = crw.buffer[:0] // Reset buffer length
	
	return err
}

// Close flushes any remaining data and cleans up resources
func (crw *ChunkedResponseWriter) Close() error {
	err := crw.flush()
	
	// Return buffer to pool
	crw.processor.PutByteSlice(crw.buffer)
	
	return err
}

// GetBytesWritten returns the total number of bytes written
func (crw *ChunkedResponseWriter) GetBytesWritten() int64 {
	return crw.written
}

// LargeDataHandler provides utilities for handling large API responses
type LargeDataHandler struct {
	processor *StreamProcessor
}

// NewLargeDataHandler creates a new large data handler
func NewLargeDataHandler(maxMemoryMB int64) *LargeDataHandler {
	return &LargeDataHandler{
		processor: NewStreamProcessor(maxMemoryMB, 16384), // 16KB chunks for large data
	}
}

// ProcessInBatches processes large datasets in memory-efficient batches
func (ldh *LargeDataHandler) ProcessInBatches(ctx context.Context, data []interface{}, batchSize int, processor func([]interface{}) error) error {
	if batchSize <= 0 {
		batchSize = 100 // Default batch size
	}
	
	var processedTotal int
	
	for i := 0; i < len(data); i += batchSize {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Check memory usage
		if !ldh.processor.CheckMemory() {
			return fmt.Errorf("memory usage exceeded limits during batch processing")
		}
		
		// Create batch
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}
		
		batch := data[i:end]
		
		// Process batch
		if err := processor(batch); err != nil {
			return fmt.Errorf("batch processing error at index %d: %w", i, err)
		}
		
		processedTotal += len(batch)
		
		// Log progress
		if processedTotal%1000 == 0 {
			allocMB, sysMB := ldh.processor.GetMemoryStats()
			log.Printf("Processed %d/%d items in batches, Memory: %dMB alloc, %dMB sys", processedTotal, len(data), allocMB, sysMB)
		}
	}
	
	log.Printf("Successfully processed %d items in batches", processedTotal)
	return nil
}

// GetMemoryStats returns current memory usage statistics
func (ldh *LargeDataHandler) GetMemoryStats() (allocMB, sysMB int64) {
	return ldh.processor.GetMemoryStats()
}