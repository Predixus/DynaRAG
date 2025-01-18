package main

import (
	"context"
	"log/slog"
	"time"

	dr "github.com/Predixus/DynaRAG"
)

func main() {
	client, err := dr.New(dr.Config{
		PostgresConnStr: "postgresql://admin:root@localhost:5053/main?sslmode=disable",
	})
	if err != nil {
		slog.Error("Unable to initialise the DynaRAG client", "message", err)
		return
	}

	start := time.Now()
	err = client.Chunk(context.Background(), "Test String", "./test", nil)
	elapsed := time.Since(start)

	if err != nil {
		slog.Error("Unable to post chunk", "message", err)
		return
	}
	slog.Info("Successfully Chunked", "duration", elapsed)

	return
}
