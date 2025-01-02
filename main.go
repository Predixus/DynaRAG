package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/Predixus/DynaRAG/middleware"
	"github.com/Predixus/DynaRAG/rag"
	"github.com/Predixus/DynaRAG/store"
	"github.com/Predixus/DynaRAG/utils"
)

//go:generate sqlc vet
//go:generate sqlc generate

const (
	rlWindow      = 100 // seconds
	rlMaxRequests = 10000
)

var (
	postgres_conn_str string
	redisUrl          string
	port              string
	http_handler      http.Handler
	once              sync.Once
)

func setup() (string, string, string) {
	var postgres_conn_str string = os.Getenv("POSTGRES_CONN_STR")
	if postgres_conn_str == "" {
		panic("`POSTGRES_CONN_STR` not set")
	}
	var redisUrl string = os.Getenv("REDIS_URL")
	if redisUrl == "" {
		panic("`REDIS_URL` not set")
	}

	var port string = os.Getenv("PORT")
	if port == "" {
		panic("`PORT` not set")
	}
	return postgres_conn_str, redisUrl, port
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Could not find `.env` file. Will obtain environment variables from system")
	}

	postgres_conn_str, redisUrl, port = setup()
}

func Stub(w http.ResponseWriter, r *http.Request) {
	log.Println("Received stub request")
	w.Write([]byte("Route not Implemented"))
}

func Chunk(w http.ResponseWriter, r *http.Request) {
	type ChunkRequestBody struct {
		Chunk    string                 `json:"chunk"`
		FilePath string                 `json:"filepath"`
		MetaData map[string]interface{} `json:"metadata,omitempty"`
	}
	userId, ok := r.Context().Value("userId").(string)
	if !ok {
		http.Error(w, "Unauthorised", http.StatusUnauthorized)
		return
	}
	chunk, err := utils.ParseJsonBody[ChunkRequestBody](w, r)
	if err != nil {
		log.Printf("Could not unmarshal json body: %v", err)
		return
	}
	_, err = store.AddEmbedding(r.Context(), userId, chunk.FilePath, chunk.Chunk, chunk.MetaData)
	if err != nil {
		log.Printf("Could not process embedding: %v", err)
		http.Error(w, "Unable to process embedding for text", http.StatusInternalServerError)
	}
}

func Similar(w http.ResponseWriter, r *http.Request) {
	type SimilarityRequest struct {
		Text string `json:"text"` // The text to compare against
		K    int8   `json:"k"`    // Number of similar results to return
	}
	userId, ok := r.Context().Value("userId").(string)
	if !ok {
		http.Error(w, "Unauthorised", http.StatusUnauthorized)
		return
	}

	req, err := utils.ParseJsonBody[SimilarityRequest](w, r)
	if err != nil {
		log.Printf("Could not unmarshal json body: %v", err)
		return
	}

	log.Println("Gathering similar documents")
	res, err := store.GetTopKEmbeddings(r.Context(), userId, req.Text, req.K)
	if err != nil {
		log.Printf("Could not get top K embeddings: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(res)
	if err != nil {
		log.Printf("Could not marshal result into json: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func Query(w http.ResponseWriter, r *http.Request) {
	type QueryRequestBody struct {
		Query string `json:"query"`
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
	res, err := store.GetTopKEmbeddings(r.Context(), userId, q.Query, 10)
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
	userId, ok := r.Context().Value("userId").(string)
	if !ok {
		http.Error(w, "Unauthorised", http.StatusBadRequest)
		return
	}
	chunks, err := store.ListUserChunks(r.Context(), userId)
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

		postgres_conn_str, redisUrl, port = setup()

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

		// setup rate limiter
		rl, err := middleware.NewRateLimiter(redisUrl, rlWindow, rlMaxRequests)
		if err != nil {
			initError = err
			return
		}

		// intialise router with all routes
		router := http.NewServeMux()
		router.HandleFunc("/", Stub)
		router.HandleFunc("POST /chunk", Chunk)
		router.HandleFunc("POST /similar", Similar)
		router.HandleFunc("POST /query", Query)
		router.HandleFunc("DELETE /chunks", DeleteChunksForUser)
		router.HandleFunc("GET /stats", GetUserStatsHandler)
		router.HandleFunc("GET /chunks", ListUserChunksHandler)

		// apply middleware stack
		middlewareStack := middleware.CreateStack(
			middleware.CORS,
			middleware.Logging,
			middleware.RateLimit(rl),
			middleware.BearerAuth,
			middleware.IncrementRequestCount,
		)

		// wrap the router with the middleware stack
		http_handler = middlewareStack(router)
	})
	return initError
}

// handler - cloud func entry point
func handler(w http.ResponseWriter, r *http.Request) {
	if err := initialiseApp(); err != nil {
		log.Printf("Failed to initialise application: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http_handler.ServeHTTP(w, r)
}

func main() {
	if err := initialiseApp(); err != nil {
		log.Fatalf("Failed to initialise: %v", err)
	}
	server := &http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: http_handler,
	}
	log.Println("Server listening on port: ", port)
	log.Fatal(server.ListenAndServe())
}
