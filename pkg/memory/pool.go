package memory

import (
	"bytes"
	"runtime"
	"sync"
)

// BytePool manages a pool of reusable byte slices for memory efficiency
type BytePool struct {
	pool sync.Pool
}

// NewBytePool creates a new byte pool with configurable buffer size
func NewBytePool(initialSize int) *BytePool {
	return &BytePool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, initialSize)
			},
		},
	}
}

// Get retrieves a byte slice from the pool
func (bp *BytePool) Get() []byte {
	return bp.pool.Get().([]byte)[:0] // Reset length but keep capacity
}

// Put returns a byte slice to the pool for reuse
func (bp *BytePool) Put(b []byte) {
	// Only return large enough slices to avoid memory fragmentation
	if cap(b) >= 1024 {
		bp.pool.Put(b)
	}
}

// BufferPool manages a pool of reusable bytes.Buffer instances
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a new buffer pool
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (bp *BufferPool) Get() *bytes.Buffer {
	buf := bp.pool.Get().(*bytes.Buffer)
	buf.Reset() // Clear any existing content
	return buf
}

// Put returns a buffer to the pool for reuse
func (bp *BufferPool) Put(buf *bytes.Buffer) {
	// Only pool buffers under a reasonable size to prevent memory bloat
	if buf.Cap() <= 64*1024 { // 64KB limit
		bp.pool.Put(buf)
	}
}

// MemoryLimiter helps control memory usage for large operations
type MemoryLimiter struct {
	maxMemoryMB    int64
	checkInterval  int
	operationCount int64
	mu             sync.Mutex
}

// NewMemoryLimiter creates a new memory limiter
func NewMemoryLimiter(maxMemoryMB int64) *MemoryLimiter {
	return &MemoryLimiter{
		maxMemoryMB:   maxMemoryMB,
		checkInterval: 100, // Check every 100 operations
	}
}

// CheckMemoryUsage checks if memory usage is within limits and optionally triggers GC
func (ml *MemoryLimiter) CheckMemoryUsage() bool {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	
	ml.operationCount++
	
	// Only check memory every N operations to avoid overhead
	if ml.operationCount%int64(ml.checkInterval) != 0 {
		return true
	}
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	currentMemoryMB := int64(m.Alloc) / (1024 * 1024)
	
	if currentMemoryMB > ml.maxMemoryMB {
		// Force garbage collection
		runtime.GC()
		
		// Check again after GC
		runtime.ReadMemStats(&m)
		currentMemoryMB = int64(m.Alloc) / (1024 * 1024)
		
		// Return false if still over limit after GC
		return currentMemoryMB <= ml.maxMemoryMB
	}
	
	return true
}

// GetMemoryStats returns current memory statistics
func (ml *MemoryLimiter) GetMemoryStats() (allocMB, sysMB int64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return int64(m.Alloc) / (1024 * 1024), int64(m.Sys) / (1024 * 1024)
}

// StreamProcessor handles streaming processing of large data to minimize memory usage
type StreamProcessor struct {
	bufferPool   *BufferPool
	bytePool     *BytePool
	memLimiter   *MemoryLimiter
	chunkSize    int
}

// NewStreamProcessor creates a new stream processor for memory-efficient processing
func NewStreamProcessor(maxMemoryMB int64, chunkSize int) *StreamProcessor {
	if chunkSize <= 0 {
		chunkSize = 8192 // 8KB default
	}
	
	return &StreamProcessor{
		bufferPool: NewBufferPool(),
		bytePool:   NewBytePool(chunkSize),
		memLimiter: NewMemoryLimiter(maxMemoryMB),
		chunkSize:  chunkSize,
	}
}

// GetBuffer returns a buffer from the pool
func (sp *StreamProcessor) GetBuffer() *bytes.Buffer {
	return sp.bufferPool.Get()
}

// PutBuffer returns a buffer to the pool
func (sp *StreamProcessor) PutBuffer(buf *bytes.Buffer) {
	sp.bufferPool.Put(buf)
}

// GetByteSlice returns a byte slice from the pool
func (sp *StreamProcessor) GetByteSlice() []byte {
	return sp.bytePool.Get()
}

// PutByteSlice returns a byte slice to the pool
func (sp *StreamProcessor) PutByteSlice(b []byte) {
	sp.bytePool.Put(b)
}

// CheckMemory checks if processing should continue based on memory usage
func (sp *StreamProcessor) CheckMemory() bool {
	return sp.memLimiter.CheckMemoryUsage()
}

// GetChunkSize returns the configured chunk size for streaming operations
func (sp *StreamProcessor) GetChunkSize() int {
	return sp.chunkSize
}

// GetMemoryStats returns current memory usage statistics
func (sp *StreamProcessor) GetMemoryStats() (allocMB, sysMB int64) {
	return sp.memLimiter.GetMemoryStats()
}