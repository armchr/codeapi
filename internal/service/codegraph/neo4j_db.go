package codegraph

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/zap"
)

// Neo4jDatabase implements the GraphDatabase interface using Neo4j
type Neo4jDatabase struct {
	driver neo4j.DriverWithContext
	logger *zap.Logger
}

// NewNeo4jDatabase creates a new Neo4j database instance
func NewNeo4jDatabase(uri, username, password string, logger *zap.Logger) (*Neo4jDatabase, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	db := &Neo4jDatabase{
		driver: driver,
		logger: logger,
	}

	return db, nil
}

// VerifyConnectivity checks if the database connection is working
func (db *Neo4jDatabase) VerifyConnectivity(ctx context.Context) error {
	return db.driver.VerifyConnectivity(ctx)
}

// Close closes the database connection
func (db *Neo4jDatabase) Close(ctx context.Context) error {
	return db.driver.Close(ctx)
}

// ExecuteRead executes a read-only Cypher query and returns the raw records
func (db *Neo4jDatabase) ExecuteRead(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	session := db.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		var records []map[string]any
		for result.Next(ctx) {
			recordMap := make(map[string]any)
			record := result.Record()

			for _, key := range record.Keys {
				value, _ := record.Get(key)
				// Convert Neo4j nodes to property maps
				if node, ok := value.(neo4j.Node); ok {
					recordMap[key] = node.GetProperties()
				} else {
					recordMap[key] = value
				}
			}
			records = append(records, recordMap)
		}

		if err = result.Err(); err != nil {
			return nil, err
		}

		return records, nil
	})

	if err != nil {
		db.logger.Error("Failed to execute read query", zap.String("query", query), zap.Error(err))
		return nil, fmt.Errorf("failed to execute read query: %w", err)
	}

	return result.([]map[string]any), nil
}

// ExecuteWrite executes a write Cypher query and returns the raw records
func (db *Neo4jDatabase) ExecuteWrite(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	session := db.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		var records []map[string]any
		for result.Next(ctx) {
			recordMap := make(map[string]any)
			record := result.Record()

			for _, key := range record.Keys {
				value, _ := record.Get(key)
				// Convert Neo4j nodes to property maps
				if node, ok := value.(neo4j.Node); ok {
					recordMap[key] = node.GetProperties()
				} else {
					recordMap[key] = value
				}
			}
			records = append(records, recordMap)
		}

		if err = result.Err(); err != nil {
			return nil, err
		}

		return records, nil
	})

	if err != nil {
		db.logger.Error("Failed to execute write query", zap.String("query", query), zap.Error(err))
		return nil, fmt.Errorf("failed to execute write query: %w", err)
	}

	return result.([]map[string]any), nil
}

// ExecuteReadSingle executes a read-only Cypher query expecting a single record
func (db *Neo4jDatabase) ExecuteReadSingle(ctx context.Context, query string, params map[string]any) (map[string]any, error) {
	records, err := db.ExecuteRead(ctx, query, params)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no records returned")
	}

	if len(records) > 1 {
		return nil, fmt.Errorf("expected single record, got %d", len(records))
	}

	return records[0], nil
}

// ExecuteWriteSingle executes a write Cypher query expecting a single record
func (db *Neo4jDatabase) ExecuteWriteSingle(ctx context.Context, query string, params map[string]any) (map[string]any, error) {
	records, err := db.ExecuteWrite(ctx, query, params)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no records returned")
	}

	if len(records) > 1 {
		return nil, fmt.Errorf("expected single record, got %d", len(records))
	}

	return records[0], nil
}

// Neo4jNode wraps a Neo4j node to implement the GraphNode interface
type Neo4jNode struct {
	node neo4j.Node
}

// GetProperties returns the node properties
func (n *Neo4jNode) GetProperties() map[string]any {
	return n.node.GetProperties()
}

// WrapNeo4jNode wraps a Neo4j node in our GraphNode interface
func WrapNeo4jNode(node neo4j.Node) GraphNode {
	return &Neo4jNode{node: node}
}
