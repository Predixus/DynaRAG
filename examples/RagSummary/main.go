package main

import (
	"context"
	"log/slog"
	"os"

	dr "github.com/Predixus/DynaRAG"
)

func main() {
	args := os.Args[1:]

	client, err := dr.New(dr.Config{
		PostgresConnStr: "postgresql://admin:root@localhost:5053/main?sslmode=disable",
		LLMProvider:     args[0],
		LLMToken:        args[1],
	})
	if err != nil {
		slog.Error("Unable to initialise the DynaRAG client", "message", err)
		return
	}

	err = client.Chunk(context.Background(), "London is the capital of england", "./test", nil, nil)
	if err != nil {
		slog.Error("Failed to chunk text", "message", err)
		return
	}

	writer := os.Stdout
	nDocs := int8(1)
	err = client.Query(
		context.Background(),
		"What is the capital of England?",
		&nDocs,
		nil,
		writer,
	)
	if err != nil {
		slog.Error("Failed to execute query", "message", err)
		return
	}
}
