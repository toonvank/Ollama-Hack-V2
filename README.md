![logo](./assets/favicon.svg)

# Ollama-Hack V2 🚀
*The Next-Generation, High-Performance Go Backend for Ollama Endpoint Aggregation!*

---

## 📖 Introduction

Ollama-Hack V2 is a complete rewrite of the original Ollama aggregator, specifically engineered in **Go** to guarantee high-performance concurrency, minimal latency, and incredible stability. 

> Many publicly exposed Ollama interfaces are available online without authentication. When you try to harness them, testing performance and verifying models individually is a massive headache. 
> 
> **Ollama-Hack V2** is your unified command center. It seamlessly manages, categorizes, tests, and load-balances public or private Ollama endpoints under one ultra-fast API roof. 

It acts as an intelligent proxy that provides a 100% **OpenAI-compatible API**, while automatically funneling your requests to the best performing underlying Ollama endpoints.

## ✨ Why V2 in Go?

We transitioned from a Python/FastAPI environment directly to **Golang** to maximize throughput. When handling LLM streams, proxying requests, and doing background task polling across dozens of endpoints, Go's goroutines ensure rock-solid stability and zero dropped connections.

V2 doesn't just proxy—it's smart, robust, and designed for heavy production usage.

## 🚀 Core Features

-   ⚡ **Go-Powered Backend**: Rewritten from the ground up in Go for extreme scalability and efficiency.
-   🔄 **Unified Endpoint Aggregation**: Centrally manage multiple Ollama endpoints. Supports batch importing!
-   ⚖️ **Smart Load Balancing**: The proxy evaluates underlying token-per-second (`token/s`) metrics to automatically select the optimal route for your OpenAI requests.
-   🧩 **100% OpenAI API Compatible**: Drop-in replacement for OpenAI endpoints in LangChain, LlamaIndex, or any common UI.
-   🔑 **API Key Generation**: Secure and distribute API Keys to clients without exposing your raw Ollama instances.
-   💰 **Custom Usage Plans**: Create tiered plans (Limits for RPM & RPD), including unlimited request plans (-1 limit)!
-   📊 **Background Performance Testing**: Built-in background polling continuously tests the health and speed of your managed nodes.
-   🎨 **Stunning React Frontend**: Polished, dark-mode ready UI built with Vite and NextUI for effortless administration.

## 🛠️ Stack & Requirements

-   **Backend**: Go 1.22+, Gin, PostgreSQL
-   **Frontend**: React, Vite, TypeScript, TailwindCSS
-   **Infrastructure**: Docker & Docker Compose (Highly Recommended)

## 🐳 Quickstart (Docker)

If you have Docker installed, you can spin up the entire ecosystem in seconds:

```bash
# Clone the repository
git clone https://github.com/timlzh/ollama-hack.git
cd ollama-hack

# The repository contains the fully composed V2 ecosystem
docker compose up -d --build
```

Visit the Admin Console: `http://localhost:3000/init` to configure your master credentials.

## 💻 Manual Development Setup

If you wish to contribute or run the components manually:

### Start the Go Backend
```bash
cd backend-go
go mod tidy
go run ./cmd/server
```
*(Ensure PostgreSQL is running and environment variables like `DATABASE_HOST` are set)*

### Start the React Frontend
```bash
cd frontend
yarn install
yarn dev
```

## 📝 OpenAI Drop-in Usage

Because the Go Proxy is fully OpenAI compatible, you can interact with it using standard cURL or OpenAI client libraries. Just pass your generated API Key!

```bash
curl -N -X POST http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_OLLAMA_HACK_API_KEY" \
  -d '{
    "model": "llama3:8b",
    "messages": [
      {"role": "system", "content": "You are a helpful proxy routing assistant."},
      {"role": "user", "content": "Hello!"}
    ],
    "temperature": 0.7,
    "stream": true
  }'
```

## ⚙️ Configuration

Configure the backend behavior via environment variables in `docker-compose.yml`:

```yaml
environment:
    - APP_ENV=prod # "dev" or "prod"
    - APP_SECRET_KEY=change_this_key # Security signing key
    - APP_ACCESS_TOKEN_EXPIRE_MINUTES=10080 # Token TTL
    - DATABASE_ENGINE=postgresql
    - DATABASE_HOST=db
    - DATABASE_PORT=5432
    - DATABASE_USERNAME=ollama_hack
    - DATABASE_PASSWORD=change_this_password
    - DATABASE_DB=ollama_hack
```

## 👤 Author & License

Originating Author: [Timlzh](https://github.com/timlzh)  
License: **MIT License**

## 🖼️ Gallery

-   Home
    ![Home](./assets/index.png)
-   Endpoint Routing
    ![Endpoint Management](./assets/endpoints.png)
-   Fleet Testing
    ![Model Management](./assets/models.png)
