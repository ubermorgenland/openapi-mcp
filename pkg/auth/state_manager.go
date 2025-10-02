package auth

import (
	"strings"
	"sync"

	"github.com/ubermorgenland/openapi-mcp/pkg/models"
)

type StateManager struct {
	specs map[string]*models.OpenAPISpec
	mutex sync.RWMutex
}

func NewStateManager() *StateManager {
	return &StateManager{
		specs: make(map[string]*models.OpenAPISpec),
	}
}

func (sm *StateManager) UpdateSpecs(specs []*models.OpenAPISpec) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	sm.specs = make(map[string]*models.OpenAPISpec)
	for _, spec := range specs {
		endpoint := strings.TrimPrefix(spec.EndpointPath, "/")
		sm.specs[endpoint] = spec
	}
}

func (sm *StateManager) GetSpec(endpoint string) (*models.OpenAPISpec, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	
	spec, exists := sm.specs[endpoint]
	return spec, exists
}