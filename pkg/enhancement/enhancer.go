package enhancement

import (
	"log"

	"github.com/gomcpgo/replicate_image_ai/pkg/client"
	"github.com/gomcpgo/replicate_image_ai/pkg/storage"
)

// Enhancer handles image enhancement operations
type Enhancer struct {
	client  *client.ReplicateClient
	storage *storage.Storage
	debug   bool
}

// NewEnhancer creates a new Enhancer instance
func NewEnhancer(client *client.ReplicateClient, storage *storage.Storage, debug bool) *Enhancer {
	return &Enhancer{
		client:  client,
		storage: storage,
		debug:   debug,
	}
}

// logDebug logs debug messages if debug mode is enabled
func (e *Enhancer) logDebug(format string, args ...interface{}) {
	if e.debug {
		log.Printf(format, args...)
	}
}