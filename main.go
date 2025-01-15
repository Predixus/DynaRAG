package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/Predixus/DynaRAG/rag"
	"github.com/Predixus/DynaRAG/store"
	"github.com/Predixus/DynaRAG/types"
	"github.com/Predixus/DynaRAG/utils"
)

//go:generate sqlc vet
//go:generate sqlc generate

var (
	postgres_conn_str string
	port              string
	http_handler      http.Handler
	once              sync.Once
)

func setup() (string, string) {
	var postgres_conn_str string = os.Getenv("POSTGRES_CONN_STR")
	if postgres_conn_str == "" {
		panic("`POSTGRES_CONN_STR` not set")
	}

	var port string = os.Getenv("PORT")
	if port == "" {
		panic("`PORT` not set")
	}
	return postgres_conn_str, port
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Could not find `.env` file. Will obtain environment variables from system")
	}

	postgres_conn_str, port = setup()
}

func Chunk(
	ctx context.Context,
	chunk string,
	filePath string,
	metadata map[string]interface{},
) error {
	_, err := store.AddEmbedding(ctx, filePath, chunk, metadata)
	if err != nil {
		slog.Error("Could not process embedding: %v", err)
		return err
	}
	return nil
}

func Similar(
	ctx context.Context,
	text string,
	k int8,
	metadata *types.JSONMap,
) ([]store.FindTopKNNEmbeddingsRow, error) {
	slog.Info("Gathering similar documents")

	res, err := store.GetTopKEmbeddings(ctx, text, k, metadata)
	if err != nil {
		slog.Error("Could not get top K embeddings: %v", err)
		return nil, err
	}
	return res, nil
}

func Query(w http.ResponseWriter, r *http.Request) {
	type QueryRequestBody struct {
		Query    string         `json:"query"`
		Metadata *types.JSONMap `json:"metadata,omitempty"`
	}
	userId, ok := r.Context().Value("userId").(string)
	if !ok {
		http.Error(w, "Unauthorised", http.StatusUnauthorized)
		return
	}

	q, err := utils.ParseJsonBody[QueryRequestBody](w, r)
	if err != nil {
		log.Printf("Could not unmarshal json body: %v", err)
		return
	}

	log.Println("Gathering similar documents")
	res, err := store.GetTopKEmbeddings(r.Context(), userId, q.Query, 10, q.Metadata)
	if err != nil {
		log.Printf("Could not get top K embeddings: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	var documents []rag.Document
	for ii, doc := range res {
		documents = append(documents, rag.Document{
			Index:   strconv.Itoa(ii),
			Source:  doc.FilePath,
			Content: doc.ChunkText,
		})
	}

	err = rag.GenerateRAGResponse(documents, q.Query, w)
	if err != nil {
		log.Println("Failed to generate RAG response: ", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func DeleteChunksForUser(w http.ResponseWriter, r *http.Request) {
	type DeleteChunksForUserRequestBody struct {
		DryRun *bool `json:"dryrun,omitempty"`
	}
	userId, ok := r.Context().Value("userId").(string)
	if !ok {
		http.Error(w, "Unauthorised", http.StatusUnauthorized)
		return
	}
	del, err := utils.ParseJsonBody[DeleteChunksForUserRequestBody](w, r)
	if err != nil {
		log.Printf("Could not unmarshal json body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	dryRun := false
	if del.DryRun != nil {
		dryRun = *del.DryRun
	}

	stats, err := store.DeleteUserEmbeddings(r.Context(), userId, dryRun)
	if err != nil {
		log.Printf("Failed to delete embeddings: %v", err)
		http.Error(w, "Unable to delete embeddings", http.StatusInternalServerError)
		return
	}

	// Return the deletion stats in the response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func GetUserStatsHandler(w http.ResponseWriter, r *http.Request) {
	type UserStatsResponse struct {
		TotalBytes    int64 `json:"total_bytes"`
		APIRequests   int64 `json:"api_requests"`
		DocumentCount int64 `json:"document_count"`
		ChunkCount    int64 `json:"chunk_count"`
	}
	userId, ok := r.Context().Value("userId").(string)
	if !ok {
		http.Error(w, "Unauthorised", http.StatusUnauthorized)
		return
	}
	stats, err := store.GetUserStats(r.Context(), userId)
	if err != nil {
		log.Printf("Failed to get user stats: %v", err)
		http.Error(w, "Unable to retrieve user stats", http.StatusInternalServerError)
		return
	}

	response := UserStatsResponse{
		TotalBytes:    stats.TotalBytes.Int64,
		APIRequests:   stats.ApiRequests,
		DocumentCount: stats.DocumentCount,
		ChunkCount:    stats.ChunkCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func ListUserChunksHandler(w http.ResponseWriter, r *http.Request) {
	type ListUserChunksRequestBody struct {
		Metadata map[string]interface{} `json:"metadata,omitempty"`
	}

	userId, ok := r.Context().Value("userId").(string)
	if !ok {
		http.Error(w, "Unauthorised", http.StatusBadRequest)
		return
	}
	var metadata *types.JSONMap
	if r.Body != nil && r.ContentLength != 0 {
		spec, err := utils.ParseJsonBody[ListUserChunksRequestBody](w, r)
		if err != nil {
			return
		}
		metadata = (*types.JSONMap)(&spec.Metadata)
	}

	chunks, err := store.ListUserChunks(r.Context(), userId, metadata)
	if err != nil {
		log.Printf("Failed to list user chunks: %v", err)
		http.Error(w, "Unable to retrieve user chunks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chunks); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func initialiseApp() error {
	var initError error

	once.Do(func() {
		// load environment variables
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found, using system environment variables")
		}

		postgres_conn_str, port = setup()

		// run migrations
		m, err := migrate.New(
			"file://store/migrations",
			postgres_conn_str,
		)
		if err != nil {
			initError = err
			return
		}

		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			initError = err
			return
		}

		if err != nil {
			initError = err
			return
		}
	})
	return initError
}
