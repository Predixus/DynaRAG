package main

import (
	"context"
	"log/slog"
	"time"

	dr "github.com/Predixus/DynaRAG"
	"github.com/Predixus/DynaRAG/types"
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
	err = client.Chunk(context.Background(), "Test String", "./test", nil, nil)
	elapsed := time.Since(start)
	start = time.Now()
	someMetadata := make(map[string]interface{}, 1)
	type SomeMetadata struct {
		FieldA string
		FieldB int16
	}
	someMetadata["EntryA"] = SomeMetadata{
		FieldA: "Hello world!",
		FieldB: 25,
	}

	err = client.Chunk(
		context.Background(),
		"Another test string!",
		"./another_directory",
		nil,
		(*types.JSONMap)(&someMetadata),
	)

	elapsed = time.Since(start)

	if err != nil {
		slog.Error("Unable to post chunk", "message", err)
		return
	}
	slog.Info("Successfully Chunked", "duration", elapsed)
	chunks, err := client.ListChunks(context.Background(), (*types.JSONMap)(&someMetadata))
	if err != nil {
		slog.Error("Unable to purge chunks", "message", err)
	}
	slog.Info("Successfully purged chunks", "data", chunks)
	return
}
