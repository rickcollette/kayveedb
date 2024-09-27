package lib

import (
	"sync"
)

type ClientManager struct {
	activeClients map[uint32]struct{} // Use a map to track active client IDs (or some identifier)
	mu            sync.Mutex
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		activeClients: make(map[uint32]struct{}),
	}
}

// Add a new client to the active client list
func (cm *ClientManager) AddClient(clientID uint32) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.activeClients[clientID] = struct{}{}
}

// Remove a client from the active client list
func (cm *ClientManager) RemoveClient(clientID uint32) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.activeClients, clientID)
}

// Get the count of active clients
func (cm *ClientManager) GetActiveClientCount() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return len(cm.activeClients)
}
