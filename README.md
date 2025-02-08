![Group 6](https://github.com/user-attachments/assets/1b37d34c-6f54-4e34-9a93-c00757377f7f)

![GitHub Release](https://img.shields.io/github/v/release/Predixus/DynaRAG)
[![Discord Widget](https://discord.com/api/guilds/1329817069146869831/widget.png)](https://discord.gg/shZeg7bYpC)
[![Go Report Card](https://goreportcard.com/badge/github.com/Predixus/DynaRAG)](https://goreportcard.com/report/github.com/Predixus/DynaRAG)

A _fast_, _robust_, and _production-ready_ RAG backend - so you can focus on the chunks.

> [!CAUTION]
> DynaRAG is in a Pre-release state. Full release and stability will arrive soon.

## Table of Contents

- [What is DynaRAG?](#what-is-dynarag)
- [Core Features](#core-features)
- [Technical Benefits](#technical-benefits)
- [About Us](#about-us)
- [Target Users](#target-users)
- [Why DynaRAG?](#why-dynarag)
- [Prerequisites](#prerequisites)
- [Module Structure](#module-structure)
  - [Feature Extraction / Embedding](#feature-extraction--embedding)
- [License](#license)

## What is DynaRAG?

DynaRAG is a RAG (Retrieval-Augmented Generation) backend that implements all the core functionality that you
would expect from a RAG framework:

- Hybrid retrieval, via [Cover Density Ranking](https://www.ra.ethz.ch/cdstore/www2002/refereed/643/node7.html) and [Vector Similarity Search](https://www.postgresql.org/docs/current/textsearch-controls.html#:~:text=Clarke%2C%20Cormack%2C%20and%20Tudhope%27s%20%22Relevance%20Ranking%20for%20One%20to%20Three%20Term%20Queries%22%20in%20the%20journal%20%22Information%20Processing%20and%20Management%22%2C%201999)
- HyDE (Hypothetical Document Embeddings) to increase retreival likelihood
- Multi-Modal Embedding, to handle image search & table based search

Given all of these methods DynaRAG focuses on providing a highly performant backend for adding, retrieving
and filtering text chunks to help you build a powerful RAG application.

DynaRAG keeps the whole process performant by by pushing the inference and database-query latencies into
the parts of the RAG pipeline that yield the lowest round trip time. There is no magic in this approach just
good engineering.

## Core Functionality

DynaRAG provides several key operations through its client interface:

- `Chunk`: Add new text chunks with associated metadata and file paths
- `Similar`: Find semantically similar chunks using vector similarity search
- `Query`: Generate RAG responses by combining relevant chunks with LLM processing
- `PurgeChunks`: Remove stored chunks (with optional dry-run)
- `GetStats`: Retrieve usage statistics
- `ListChunks`: List all stored chunks with their metadata

During initialisation (`client.Initialise()`), DynaRAG automatically runs database migrations to:

1. Set up the required PostgreSQL extensions (pgvector)
2. Create necessary tables for storing embeddings and metadata
3. Configure indexes for efficient vector similarity search

These migrations ensure your database is properly configured for vector operations and chunk
storage with DynaRAG. The migrations are managed using `golang-migrate`.

> [!NOTE]
> The database must be accessible with the provided connection string and the user must have
> sufficient privileges to create extensions and tables.

## Technical Benefits

DynaRAG is written entirely in Go, including feature extraction models interfaced with via the Onnx
runtime. This provides several advantages over Python-based approaches:

- Inherently faster performance, as we can leverage Golangs awesome concurrency model
- Single binary compilation for easier deployment
- Generally, better memory safety
- No performance loss in HTTP layer communication with feature extraction service
- Strongly typed from the offset
- Easily slots in to applications that Golang was made for (i.e. servers), without a networking overhead

## About Us

DynaRAG is built and maintained by [Predixus](https://www.predixus.com), an Analytics and Data
company based in Cambridge, UK.

## Target Users

DynaRAG is ideal for developers or product owners looking to add RAG capabilities to their
applications in a lightweight and performant manner. It excels when working with clear text
chunks that directly represent potential answers to user questions.

## Why DynaRAG?

DynaRAG was developed to address the need for a simple, self-hosted RAG solution for internal
and client projects. DynaRAG was born out of our need for a fast, robust and simple
RAG backend that didn't break the bank and allowed us to own the inference resources.

The key considerations to the project were:

- Minimal project footprint
- Cost-effective implementation
- Focus on optimal chunking rather than complex retrieval
- Ability to own inference capability and make the most use of compute resources available

> [!TIP]
> Focus on the quality of your text chunks when using DynaRAG. If each chunk clearly represents
> an answer to likely questions, naive RAG becomes highly effective.

## Prerequisites

DynaRAG depends on the Onnx runtime to run the embedding pipelines. Ensure that the runtime is
present in the default directory (`/usr/lib/onnxruntime.so`). It can be downloaded from the
Microsoft [downloads page](https://github.com/microsoft/onnxruntime/releases).

The Go bindings for Huggingface tokenisers is also required. Download it from [this repo](https://github.com/daulet/tokenizers/releases) and
place it in the default location `/usr/lib/tokenizers.a`.

## Module Structure

The DynaRAG Go! module is split into several packages:

- `internal/llm` - defines code to interface directly with the LLM provider (Groq, Ollama etc.)
- `internal/store` - the interface to the PGVector store. The home of the sqlc auto-generated code and the migrations
  managed by `go-migrate`
- `internal/embed` - the embedding process powered by [Hugot](https://github.com/knights-analytics/hugot)
- `internal/rag` - code that defines the final summarisation layer, along with system prompts
- `types` - globally used types, some of which are used by `sqlc` during code generation
- `internal/utils` - miscellaneous utilities
- `migrations` - contains the Postgres migrations required to configure your postgres instance for DynaRAG
  `query.sql` defines raw pSQL queries that drive the interactions with the PGVector instance.

### Feature Extraction / Embedding

Feature extraction (conversion of the text chunks into vectors) is performed through Hugot via the
Onnx runtime.

On inference, the required models will be downloaded from Huggingface and stored in the ./models/
folder. This will only be done once to obtain the relevant .onnx binaries.

## License

DynaRAG is licensed under the BSD 3-Clause License. See [LICENSE](LICENSE) for details.
