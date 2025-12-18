package vector

import (
	"github.com/armchr/codeapi/internal/model"
	"github.com/armchr/codeapi/pkg/lsp/base"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"go.uber.org/zap"
)

// QdrantDatabase implements VectorDatabase interface using Qdrant
type QdrantDatabase struct {
	client *qdrant.Client
	logger *zap.Logger
}

// NewQdrantDatabase creates a new Qdrant database connection
func NewQdrantDatabase(host string, port int, apiKey string, logger *zap.Logger) (*QdrantDatabase, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   host,
		Port:   port,
		APIKey: apiKey,
		UseTLS: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	return &QdrantDatabase{
		client: client,
		logger: logger,
	}, nil
}

// CreateCollection creates a new collection with the specified dimension and distance metric
func (q *QdrantDatabase) CreateCollection(ctx context.Context, collectionName string, vectorDim int, distance DistanceMetric) error {
	// Map our distance metric to Qdrant's distance type
	var qdrantDistance qdrant.Distance
	switch distance {
	case DistanceMetricCosine:
		qdrantDistance = qdrant.Distance_Cosine
	case DistanceMetricDot:
		qdrantDistance = qdrant.Distance_Dot
	case DistanceMetricEuclidean:
		qdrantDistance = qdrant.Distance_Euclid
	default:
		qdrantDistance = qdrant.Distance_Cosine
	}

	err := q.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(vectorDim),
			Distance: qdrantDistance,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	q.logger.Info("Created Qdrant collection", zap.String("collection", collectionName), zap.Int("dim", vectorDim))
	return nil
}

// DeleteCollection deletes a collection
func (q *QdrantDatabase) DeleteCollection(ctx context.Context, collectionName string) error {
	err := q.client.DeleteCollection(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	return nil
}

// CollectionExists checks if a collection exists
func (q *QdrantDatabase) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	exists, err := q.client.CollectionExists(ctx, collectionName)
	if err != nil {
		return false, fmt.Errorf("failed to check collection existence: %w", err)
	}
	return exists, nil
}

// UpsertChunks inserts or updates code chunks in the vector database
func (q *QdrantDatabase) UpsertChunks(ctx context.Context, collectionName string, chunks []*model.CodeChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	points := make([]*qdrant.PointStruct, 0, len(chunks))

	for _, chunk := range chunks {
		if len(chunk.Embedding) == 0 {
			q.logger.Warn("Skipping chunk without embedding", zap.String("id", chunk.ID))
			continue
		}

		// Convert CodeChunk to Qdrant point
		// Note: content is excluded to save storage space - use file_path and line numbers to retrieve content
		point := &qdrant.PointStruct{
			Id: qdrant.NewIDUUID(chunk.ID),
			Vectors: qdrant.NewVectorsMap(map[string]*qdrant.Vector{
				"": qdrant.NewVector(chunk.Embedding...),
			}),
			Payload: qdrant.NewValueMap(map[string]any{
				"chunk_type":  string(chunk.ChunkType),
				"level":       chunk.Level,
				"parent_id":   chunk.ParentID,
				"language":    chunk.Language,
				"file_path":   chunk.FilePath,
				"start_line":  chunk.StartLine,
				"end_line":    chunk.EndLine,
				"range":       rangeToMap(chunk.Range),
				"name":        chunk.Name,
				"signature":   chunk.Signature,
				"docstring":   chunk.Docstring,
				"module_name": chunk.ModuleName,
				"class_name":  chunk.ClassName,
				"metadata":    chunk.Metadata,
			}),
		}
		points = append(points, point)
	}

	if len(points) == 0 {
		q.logger.Warn("No points to upsert after filtering", zap.String("collection", collectionName))
		return nil
	}

	// Log details before upsert
	q.logger.Debug("Attempting upsert",
		zap.String("collection", collectionName),
		zap.Int("points_count", len(points)))

	_, err := q.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         points,
	})
	if err != nil {
		q.logger.Error("Upsert failed",
			zap.String("collection", collectionName),
			zap.Error(err))
		return fmt.Errorf("failed to upsert chunks: %w", err)
	}

	q.logger.Info("Upserted chunks to Qdrant",
		zap.String("collection", collectionName),
		zap.Int("count", len(points)))
	return nil
}

// SearchSimilar finds similar code chunks using vector similarity search
func (q *QdrantDatabase) SearchSimilar(ctx context.Context, collectionName string, queryVector []float32, limit int, filter map[string]interface{}) ([]*model.CodeChunk, []float32, error) {
	// Build Qdrant filter if provided
	var qdrantFilter *qdrant.Filter
	if len(filter) > 0 {
		conditions := make([]*qdrant.Condition, 0, len(filter))
		for key, value := range filter {
			conditions = append(conditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key:   key,
						Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: fmt.Sprint(value)}},
					},
				},
			})
		}
		qdrantFilter = &qdrant.Filter{
			Must: conditions,
		}
	}

	searchResult, err := q.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collectionName,
		Query:          qdrant.NewQuery(queryVector...),
		Limit:          qdrant.PtrOf(uint64(limit)),
		Filter:         qdrantFilter,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to search: %w", err)
	}

	chunks := make([]*model.CodeChunk, 0, len(searchResult))
	scores := make([]float32, 0, len(searchResult))

	for _, point := range searchResult {
		chunk := pointToCodeChunk(point)
		if chunk != nil {
			chunks = append(chunks, chunk)
			scores = append(scores, point.Score)
		}
	}

	return chunks, scores, nil
}

// GetChunkByID retrieves a specific chunk by its ID
func (q *QdrantDatabase) GetChunkByID(ctx context.Context, collectionName string, chunkID string) (*model.CodeChunk, error) {
	points, err := q.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: collectionName,
		Ids:            []*qdrant.PointId{qdrant.NewIDUUID(chunkID)},
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("chunk not found: %s", chunkID)
	}

	return retrievedPointToCodeChunk(points[0]), nil
}

// DeleteChunk deletes a chunk by its ID
func (q *QdrantDatabase) DeleteChunk(ctx context.Context, collectionName string, chunkID string) error {
	_, err := q.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{qdrant.NewIDUUID(chunkID)},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete chunk: %w", err)
	}
	return nil
}

// GetChunksByFilePath retrieves all chunks for a specific file path
func (q *QdrantDatabase) GetChunksByFilePath(ctx context.Context, collectionName string, filePath string) ([]*model.CodeChunk, error) {
	// Build filter for file_path
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key:   "file_path",
						Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: filePath}},
					},
				},
			},
		},
	}

	// Scroll through all points matching the filter
	// Using a large limit to get all chunks for a file (unlikely to have >10000 chunks in one file)
	// Note: We DO need vectors here because we reuse embeddings from existing chunks
	scrollResult, err := q.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: collectionName,
		Filter:         filter,
		Limit:          qdrant.PtrOf(uint32(10000)),
		WithPayload:    qdrant.NewWithPayload(true),
		WithVectors:    qdrant.NewWithVectors(true), // Required: we reuse these embeddings
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scroll points: %w", err)
	}

	chunks := make([]*model.CodeChunk, 0, len(scrollResult))
	for _, point := range scrollResult {
		chunk := retrievedPointToCodeChunk(point)
		if chunk != nil {
			chunks = append(chunks, chunk)
		}
	}

	return chunks, nil
}

// Close closes the database connection
func (q *QdrantDatabase) Close() error {
	if q.client != nil {
		return q.client.Close()
	}
	return nil
}

// Health checks the health of the vector database
func (q *QdrantDatabase) Health(ctx context.Context) error {
	_, err := q.client.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	return nil
}

// Helper functions

func rangeToMap(r base.Range) map[string]interface{} {
	return map[string]interface{}{
		"start": map[string]interface{}{
			"line":      r.Start.Line,
			"character": r.Start.Character,
		},
		"end": map[string]interface{}{
			"line":      r.End.Line,
			"character": r.End.Character,
		},
	}
}

func mapToRange(m map[string]interface{}) base.Range {
	start := m["start"].(map[string]interface{})
	end := m["end"].(map[string]interface{})

	return base.Range{
		Start: base.Position{
			Line:      int(start["line"].(float64)),
			Character: int(start["character"].(float64)),
		},
		End: base.Position{
			Line:      int(end["line"].(float64)),
			Character: int(end["character"].(float64)),
		},
	}
}

func pointToCodeChunk(point *qdrant.ScoredPoint) *model.CodeChunk {
	payload := point.GetPayload()
	return payloadToCodeChunk(point.Id.GetUuid(), payload)
}

func retrievedPointToCodeChunk(point *qdrant.RetrievedPoint) *model.CodeChunk {
	payload := point.GetPayload()
	return payloadToCodeChunk(point.Id.GetUuid(), payload)
}

func payloadToCodeChunk(id string, payload map[string]*qdrant.Value) *model.CodeChunk {
	if payload == nil {
		return nil
	}

	// Parse UUID to string if needed
	chunkID := id
	if parsedUUID, err := uuid.Parse(id); err == nil {
		chunkID = parsedUUID.String()
	}

	chunk := &model.CodeChunk{
		ID:         chunkID,
		ChunkType:  model.ChunkType(getStringValue(payload, "chunk_type")),
		Level:      int(getIntValue(payload, "level")),
		ParentID:   getStringValue(payload, "parent_id"),
		Content:    getStringValue(payload, "content"),
		Language:   getStringValue(payload, "language"),
		FilePath:   getStringValue(payload, "file_path"),
		StartLine:  int(getIntValue(payload, "start_line")),
		EndLine:    int(getIntValue(payload, "end_line")),
		Name:       getStringValue(payload, "name"),
		Signature:  getStringValue(payload, "signature"),
		Docstring:  getStringValue(payload, "docstring"),
		ModuleName: getStringValue(payload, "module_name"),
		ClassName:  getStringValue(payload, "class_name"),
	}

	// Parse range
	if rangeValue, ok := payload["range"]; ok {
		if rangeStruct := rangeValue.GetStructValue(); rangeStruct != nil && rangeStruct.Fields != nil {
			rangeMap := structToMap(rangeStruct)
			chunk.Range = mapToRange(rangeMap)
		}
	}

	// Parse metadata
	if metadataValue, ok := payload["metadata"]; ok {
		if metadataStruct := metadataValue.GetStructValue(); metadataStruct != nil && metadataStruct.Fields != nil {
			chunk.Metadata = structToMap(metadataStruct)
		}
	}

	return chunk
}

func getStringValue(payload map[string]*qdrant.Value, key string) string {
	if val, ok := payload[key]; ok {
		return val.GetStringValue()
	}
	return ""
}

func getIntValue(payload map[string]*qdrant.Value, key string) int64 {
	if val, ok := payload[key]; ok {
		return val.GetIntegerValue()
	}
	return 0
}

func structToMap(s *qdrant.Struct) map[string]interface{} {
	result := make(map[string]interface{})
	if s == nil || s.Fields == nil {
		return result
	}

	for key, value := range s.Fields {
		switch v := value.Kind.(type) {
		case *qdrant.Value_StringValue:
			result[key] = v.StringValue
		case *qdrant.Value_IntegerValue:
			result[key] = float64(v.IntegerValue)
		case *qdrant.Value_DoubleValue:
			result[key] = v.DoubleValue
		case *qdrant.Value_BoolValue:
			result[key] = v.BoolValue
		case *qdrant.Value_StructValue:
			result[key] = structToMap(v.StructValue)
		}
	}
	return result
}
