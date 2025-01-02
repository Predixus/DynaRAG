# DynaRAG

![GitHub Release](https://img.shields.io/github/v/release/Predixus/DynaRAG)

A _fast_, _dynamic_, and _production-ready_ RAG as a service.

> [!CAUTION]
> DynaRAG is in a Pre-release state. Full release and stability will arrive soon.

## Table of Contents
- [What is DynaRAG?](#what-is-dynarag)
- [Core Features](#core-features)
- [Technical Benefits](#technical-benefits)
- [About Us](#about-us)
- [Target Users](#target-users)
- [Why DynaRAG?](#why-dynarag)
- [Getting Started](#getting-started)
  - [Managed Service](#managed-service)
  - [Docker Container](#docker-container)
  - [Run from Source](#run-from-source)
- [Development Guide](#development-guide)
  - [Prerequisites](#prerequisites)
  - [Initial Setup](#initial-setup)
  - [Running the API](#running-the-api)
  - [Making Requests](#making-requests)

## What is DynaRAG?

DynaRAG is a RAG (Retrieval-Augmented Generation) as a service that implements a naive approach.
It focuses on providing a highly performant service that allows quick addition of text chunks to
a vector store, with optional LLM summarisation capabilities.

## Core Features

- Naive RAG with Go-managed feature extraction models (props to
[hugot](https://github.com/knights-analytics/hugot) for building awesome go Onnz bindings)
- PGVector based vector store
- Rate limiting via Redis
- Chunk separation by `UserId`
- LLM provider integration for summarisation:
  - Groq
  - OpenAI (coming in 2025)
  - Anthropic (coming in 2025)
  - Ollama (coming in 2025)

## Technical Benefits

DynaRAG is written entirely in Go, including feature extraction models interfaced with via the Onnx
runtime. This provides several advantages over Python-based approaches:

- Inherently faster performance
- Single binary compilation for easier deployment
- Enhanced memory safety for large data throughput in cloud services
- No performance loss in HTTP layer communication with feature extraction service

## About Us

DynaRAG is built and maintained by [Predixus](https://www.predixus.com), an Analytics and Data
company based in Cambridge, UK.

## Target Users

DynaRAG is ideal for developers or product owners looking to add RAG capabilities to their
applications in a lightweight and performant manner. It excels when working with clear text
chunks that directly represent potential answers to user questions.

## Why DynaRAG?

DynaRAG was developed to address the need for a simple, self-hosted RAG solution for internal
and client projects. Key considerations include:

- Minimal project footprint
- Cost-effective implementation
- Focus on optimal chunking rather than complex retrieval

> [!TIP]
> Focus on the quality of your text chunks. If each chunk clearly represents
an answer to likely questions, naive RAG becomes highly effective.

## Getting Started

Choose the deployment option that best suits your needs:

### Managed Service
- Coming in 2025

### Docker Container
- Coming soon

### Run from Source
- Follow the development guide below

## API Docs
- Coming Soon.

## Development Guide

### Prerequisites

- `cmake`
- Docker with `docker compose`
- [`sqlc`](https://docs.sqlc.dev/en/stable/overview/install.html)
- [`air`](https://github.com/air-verse/air)

### Initial Setup

1. Start PGVector and Redis:
```bash
docker compose up -d
```

2. Configure environment:
```bash
cp ./.env.example ./.env
```

3. Update environment variables:
```env
POSTGRES_CONN_STR=postgresql://admin:root@localhost:5053/main
REDIS_URL=redis://:53c2b86b1b3e8e91ac502c54cf49fcfd91e7d1271130b4de@localhost:6380
```

4. Install required binaries:
```bash
make setup
```

> [!NOTE]
> Binary builds are Linux-specific. MacOS and Windows users need to build
> Onnx and Tokenizer binaries manually.

### Running the API

Development mode:
```bash
air
```

Or build and run directly:
```bash
go build main.go
./main
```
### JWT Bearer Token
DynaRAG currently requires a [jwt](
https://jwt.io/#debugger-io?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJmcmVkIiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjk5OTk5OTk5OTksIm5iZiI6MTUxNjIzOTAyMX0.XQhc2JJvw7llZlNbN22ifaEsYHmKbmlsyF4yNqx_XYE)
token as Bearer Authorization. The token must have the
following structure:

*Header*
```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```
*Payload*
```json
{
  "sub": "test_user_1234",  # this is the UserID
  "iat": 1516239022,        # time of issue
  "exp": 9999999999,        # when the token expires
  "nbf": 1516239021         # when the token is valid from
}
```

You must then sign your JWT with the secret stored in the `JWT_SECRET` environment variable.

### Making Requests

Via curl:
```bash
curl -X GET "http://localhost:7890/stats" \
-H "Authorization: Bearer <your_signed_jwt>" \
-H "Accept: */*" \
-H "Connection: keep-alive"
```

Additional options:
- Use [Postman](https://predixus.postman.co/workspace/Predixus~6a7e467f-45da-4e1d-8583-cc2611bf0431/collection/35165780-5ace5502-2a05-4179-a0c8-ff27dba0df9b?action=share&creator=35165780)
- Use the official [Python Client](https://github.com/Predixus/DynaRAG-Python-Client)

## License

DynaRAG is licensed under the BSD 3-Clause License. See [LICENSE](LICENSE) for details.

