package codegraph

import (
	"context"
)

// GraphDatabase represents a generic graph database interface for executing Cypher queries
type GraphDatabase interface {
	// ExecuteRead executes a read-only Cypher query and returns the raw records
	ExecuteRead(ctx context.Context, query string, params map[string]any) ([]map[string]any, error)

	// ExecuteWrite executes a write Cypher query and returns the raw records
	ExecuteWrite(ctx context.Context, query string, params map[string]any) ([]map[string]any, error)

	// ExecuteReadSingle executes a read-only Cypher query expecting a single record
	ExecuteReadSingle(ctx context.Context, query string, params map[string]any) (map[string]any, error)

	// ExecuteWriteSingle executes a write Cypher query expecting a single record
	ExecuteWriteSingle(ctx context.Context, query string, params map[string]any) (map[string]any, error)

	// Close closes the database connection
	Close(ctx context.Context) error

	// VerifyConnectivity checks if the database connection is working
	VerifyConnectivity(ctx context.Context) error
}

// GraphNode represents a node returned from the graph database
type GraphNode interface {
	GetProperties() map[string]any
}
