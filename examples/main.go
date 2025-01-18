package main

import (
	"context"
	"log/slog"

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
	err = client.Chunk(context.Background(), "Test String", "./test", nil)
	if err != nil {
		slog.Error("Unable to post chunk", "message", err)
		return
	}
	return
}
