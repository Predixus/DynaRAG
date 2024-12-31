# DynaRAG
A _fast_, _dynamic_, and _production ready_ RAG as a service.

## What is it?
DynaRAG is a RAG as a service that implements a naive approach. The focus of DynaRAG is to provide a very
performant service that allows chunks of text to be added to a vector store quickly, and then queried, with 
or without LLM summarisation.

The core features of DynaRAG are:

- Naive RAG with go managed feature extraction models 
- Data storage via PGVector
- Rate limiting via Redis
- Data separation by `UserId`
- Integration with following LLM providers for the summarisation layer:
    - Groq
    - OpenAI (arriving in 2025)
    - Anthropic (arriving in 2025)

DynaRAG is written completely in Go!, including the feature extraction models which are interfaced with via
the Onnx runtime. This has some advantages over a Python based approach:

- It's inherently faster
- Compiles down to a single binary, which make deploying easier
- Higher level of memory safety, which is important to large data throughput when hosting on cloud services
- No performance loss in a HTTP layer communicating to a feature extraction service

## By Whom?
DynaRAG is built and maintained by [Predixus](https://www.predixus.com) - an Analytics and Data company
based in Cambridge, UK.

## For Who?
DynaRAG is for anyone that is looking to add RAG capabilities to their application, in a lightweight and 
performant manner. DynaRAG will excel if you have clear chunks of text that clearly represent a potential
answer to a user question, and you simply want to wrap these in RAG capability.

## Why?
DynaRAG was built out of need to have a simple self hosted RAG solution for internal and client projects 
that left a minimal footprint on the project. It was built out of the need for a fast and dynamic service
that didn't break the bank.

It was also clear that when building RAG based solutions, there is no free lunch. *But* you do get to decide
where you want the complexity. Do you want the complexity in the retreival stage, or in the data conditioning
stage when you parse documents and construct the chunks? Well, it seems like the latter approach will give
the best runtime performance, because if you nail the chunking stage so each chunk clearly represents an 
'answer' to a question then naive RAG is all that's required, which is more performant than the more complex
RAG approaches. This is why the core interface to DynaRAG is to just provide 'chunks' of text. 

> [!TIP]
> Depending on the complexity of your RAG project, focus on the quality of your text chunks. If each chunk
> is *very* representative of an answer to the kind of questions that would be asked, then naive RAG is all
> you need.

## Getting Started
DynaRAG should be simple to setup so you can get going fast. So here are some options, in increasing complexity, on
how you can run DynaRAG and start wrapping your applications in a RAG service. 

### Use the Managed Service
Arriving in 2025.

### Run it in a Docker Container
Arriving imminently.

### Run from source
Follow guide in the Developing section to get DynaRAG running locally from source.

## Developing
The following guide assumes you are developing on a linux based system (or WSL), as it obtains runtimes
from the release portion of this repo which are linux specific.

If you are developing on MacOS or Windows, you will need to build the Onnx and tokenizer runtimes
from scratch.

To build the Onnx runtime, see the [following guide](https://onnxruntime.ai/docs/build/) and to build 
the go bindings for HuggingFace tokenizers, see the instructions in the [source repo](https://github.com/daulet/tokenizers).

### Setting the Stage
Prerequisites:
- `cmake`
- Docker, incl. `docker compose`
- [`sqlc`](https://docs.sqlc.dev/en/stable/overview/install.html)
- [`air`](https://github.com/air-verse/air)

First, initialise a PGVector store and redis instance locally, using the `docker-compose.yml` file:

```bash
docker compose up -d
```

Copy over the `.env.example` file:
```bash
cp ./.env.example ./.env
```

Set the Postgres and Redis URLs to match the configuration in the docker compose:

```env
POSTGRES_CONN_STR=postgresql://admin:root@localhost:5053/main
REDIS_URL=redis://:53c2b86b1b3e8e91ac502c54cf49fcfd91e7d1271130b4de@localhost:6380
```
Complete the other requirement variables.

Now, grab the binaries used by the feature extraction model(s):
```bash
make setup
```

> [!NOTE]
> The binaries are built for Linux. If you are using either MacOS or Windows, you will need to build
> the Onnx and Tokenizer binaries yourself.

### Run the API
Once you have all of this in place, you should be good to go. Run the API in dev mode by running `air`.

Or, build the app binary and run it directly:
```bash
go build main.go
./main
```

You can then make direct Http request to your DynaRAG instance like so:
```bash
curl -X GET "http://localhost:7890/stats" \
-H "Authorization: Bearer <your_signed_jwt>" \
-H "Accept: */*" \
-H "User-Agent: PostmanRuntime/7.43.0" \
-H "Connection: keep-alive"
```

Or, via a API client like [Postman](https://predixus.postman.co/workspace/Predixus~6a7e467f-45da-4e1d-8583-cc2611bf0431/collection/35165780-5ace5502-2a05-4179-a0c8-ff27dba0df9b?action=share&creator=35165780).

You can also interact directly with your DynaRAG service via the 
[Python Client](https://github.com/Predixus/DynaRAG-Python-Client).
