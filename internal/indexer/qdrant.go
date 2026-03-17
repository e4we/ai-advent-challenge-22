// Пакет indexer реализует клиент к векторной базе данных Qdrant через gRPC.
// Отвечает за создание коллекций, запись чанков и семантический поиск.
package indexer

import (
	"context"
	"fmt"
	"log/slog"
	"rag-pipeline/internal/models"

	"github.com/qdrant/go-client/qdrant"
)

// QdrantIndexer управляет одной коллекцией в Qdrant.
type QdrantIndexer struct {
	client         *qdrant.Client // gRPC-клиент Qdrant; создаётся в NewQdrantIndexer
	CollectionName string         // имя коллекции ("rag_fixed" или "rag_structural")
}

// NewQdrantIndexer создаёт QdrantIndexer с gRPC-подключением к Qdrant.
func NewQdrantIndexer(host string, port int, collection string) (*QdrantIndexer, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to Qdrant at %s:%d: %w\nHint: run 'docker compose up -d' to start Qdrant", host, port, err)
	}
	return &QdrantIndexer{
		client:         client,
		CollectionName: collection,
	}, nil
}

// CreateCollection deletes if exists and creates a new collection with Cosine distance.
func (q *QdrantIndexer) CreateCollection(ctx context.Context, vectorSize uint64) error {
	exists, err := q.client.CollectionExists(ctx, q.CollectionName)
	if err != nil {
		return fmt.Errorf("checking collection existence: %w", err)
	}
	if exists {
		if err := q.client.DeleteCollection(ctx, q.CollectionName); err != nil {
			return fmt.Errorf("deleting collection %s: %w", q.CollectionName, err)
		}
		slog.Info("deleted existing collection", "name", q.CollectionName)
	}

	err = q.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: q.CollectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     vectorSize,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("creating collection %s: %w", q.CollectionName, err)
	}

	slog.Info("created collection", "name", q.CollectionName, "vector_size", vectorSize)
	return nil
}

// upsertBatchSize — количество точек в одном gRPC-запросе Upsert.
// Значение 100 подобрано под лимиты gRPC: не превышает типичный размер сообщения,
// но достаточно велико для эффективной передачи данных.
const upsertBatchSize = 100

// Upsert inserts or updates chunks in the collection in batches.
func (q *QdrantIndexer) Upsert(ctx context.Context, chunks []models.Chunk) error {
	for i := 0; i < len(chunks); i += upsertBatchSize {
		end := i + upsertBatchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		batch := chunks[i:end]

		points := make([]*qdrant.PointStruct, 0, len(batch))
		for _, ch := range batch {
			pointID := models.GeneratePointID(ch.Metadata.Source, ch.Metadata.Strategy, ch.Metadata.ChunkIndex)

			payload := qdrant.NewValueMap(map[string]any{
				"source":   ch.Metadata.Source,
				"title":    ch.Metadata.Title,
				"section":  ch.Metadata.Section,
				"strategy": ch.Metadata.Strategy,
				"chunk_id": ch.Metadata.ChunkID,
				"text":     ch.Text,
			})

			points = append(points, &qdrant.PointStruct{
				Id:      qdrant.NewIDNum(pointID),
				Vectors: qdrant.NewVectors(ch.Embedding...),
				Payload: payload,
			})
		}

		_, err := q.client.Upsert(ctx, &qdrant.UpsertPoints{
			CollectionName: q.CollectionName,
			Points:         points,
		})
		if err != nil {
			return fmt.Errorf("upserting batch %d-%d: %w", i, end, err)
		}

		slog.Info("upserted batch", "collection", q.CollectionName, "from", i, "to", end)
	}

	return nil
}

// Search finds the top-k nearest neighbors.
func (q *QdrantIndexer) Search(ctx context.Context, vector []float32, topK int) ([]models.SearchResult, error) {
	topKUint := uint64(topK)
	results, err := q.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: q.CollectionName,
		Query:          qdrant.NewQuery(vector...),
		Limit:          &topKUint,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("searching collection %s: %w", q.CollectionName, err)
	}

	searchResults := make([]models.SearchResult, 0, len(results))
	for _, r := range results {
		payload := r.Payload

		text := ""
		if v, ok := payload["text"]; ok {
			text = v.GetStringValue()
		}

		chunk := models.Chunk{
			Text: text,
			Metadata: models.ChunkMetadata{
				Source:    getStringPayload(payload, "source"),
				Title:     getStringPayload(payload, "title"),
				Section:   getStringPayload(payload, "section"),
				Strategy:  getStringPayload(payload, "strategy"),
				ChunkID:   getStringPayload(payload, "chunk_id"),
				CharCount: len(text),
			},
		}

		searchResults = append(searchResults, models.SearchResult{
			Chunk: chunk,
			Score: r.Score,
		})
	}

	return searchResults, nil
}

// getStringPayload безопасно извлекает строковое значение из payload Qdrant по ключу.
// Возвращает пустую строку, если ключ отсутствует — это нормально для необязательных полей.
func getStringPayload(payload map[string]*qdrant.Value, key string) string {
	if v, ok := payload[key]; ok {
		return v.GetStringValue()
	}
	return ""
}

// CollectionInfo returns the number of points in the collection.
func (q *QdrantIndexer) CollectionInfo(ctx context.Context) (uint64, error) {
	info, err := q.client.GetCollectionInfo(ctx, q.CollectionName)
	if err != nil {
		return 0, fmt.Errorf("getting collection info for %s: %w", q.CollectionName, err)
	}
	return info.GetPointsCount(), nil
}

// Close closes the gRPC connection.
func (q *QdrantIndexer) Close() error {
	return q.client.Close()
}
