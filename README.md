![logo](./assets/favicon.svg)

# Ollama-Hack V2 🚀

## 📖 Introduction

> Many publicly exposed Ollama interfaces are available online without authentication. You want to use them, but testing their performance and checking available models one by one is too tedious. Plus, you might need to frequently switch between failing interfaces.
>
> Try Ollama-Hack! It's a Python-based aggregation platform that helps you easily manage, test, and seamlessly use multiple Ollama interfaces.

Ollama-Hack is a service for managing, testing, and forwarding Ollama APIs. It centrally manages multiple Ollama endpoints and automatically selects the optimal route based on performance, providing an OpenAI-compatible API. The platform offers a friendly web interface for managing endpoints, models, API keys, and usage plans.

## ✨ Features

-   🔄 **Multi-Endpoint Management**: Centrally manage multiple Ollama service endpoints with batch import support
    ![Endpoint Management](./assets/endpoints.png)
-   🔍 **Endpoint Details**: View detailed information and available models for each endpoint
    ![Endpoint Details](./assets/endpoint_details.png)
-   🧩 **OpenAI API Compatible**: Provides OpenAI-compatible API interface
-   ⚖️ **Optimal Route Selection**: Automatically selects the best Ollama endpoint based on Token/s performance
-   🔑 **API Key Management**: Generate and manage API keys for authentication
-   📊 **Performance Monitoring**: Test and display performance metrics for models on different endpoints
-   📝 **Model Management**: Search and view available models
    ![Model Management](./assets/models.png)
-   📈 **Model Performance**: View detailed performance data for each model
    ![Model Details](./assets/model_details.png)
-   🔐 **User Management**: Admins can create and manage user accounts
-   💰 **Plan Management**: Create and manage different usage plans with API request rate limits
-   🌙 **Dark Mode**: Supports light/dark theme switching

## 🛠️ Requirements

-   Docker and Docker Compose (recommended)
-   Or Python 3.12+ (for direct execution)

## 🚀 Installation & Deployment

### Method 1: Docker Deployment (Recommended)

If you have Docker and Docker Compose installed, start with a single command:

```bash
# Download docker-compose.yml file
curl -o docker-compose.yml https://raw.githubusercontent.com/timlzh/ollama-hack/main/docker-compose.example.yml

# Modify sensitive configurations like secret keys in docker-compose.yml
vim docker-compose.yml

# Start services
docker compose up -d
```

After starting, visit http://localhost:3000/init to use the platform.

### Method 2: Direct Execution (Development Environment)

#### Backend

```bash
cd backend
# Install dependencies using Poetry
pip install poetry
poetry install

# Start service
poetry run uvicorn src.main:app --host 0.0.0.0 --port 8000
```

#### Frontend

```bash
cd frontend
# Install dependencies
yarn install

# Start in development mode
yarn dev
```

## 📝 Usage

### Web Interface

Visit http://localhost:3000/init to initialize the admin account.

After logging in, you can:

-   Create and manage user accounts
-   Add and manage Ollama endpoints
-   Generate API keys
-   Create and assign usage plans
-   View model performance data

### Plan Management

V2 adds plan management functionality. Admins can create different usage plans and assign them to users. Each plan can set:

-   Requests per minute limit (RPM)
-   Requests per day limit (RPD)
-   Default plan flag

### API Usage Examples

#### Ollama API Compatible

```bash
curl -N -X POST http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "llama3",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant"},
      {"role": "user", "content": "Hello, please introduce yourself"}
    ],
    "temperature": 0.7,
    "stream": true
  }'
```

Ollama-Hack supports all Ollama OpenAI-compatible APIs. For details, see: [Ollama/OpenAI Compatibility](https://github.com/ollama/ollama/blob/main/docs/openai.md).

## 🔧 Configuration Options

### Environment Variables

In the docker-compose.yml file, you can customize the backend through these environment variables:

```yaml
environment:
    - APP__ENV=prod # Environment type: dev or prod
    - APP__LOG_LEVEL=INFO # Log level
    - APP__SECRET_KEY=change_this_key # JWT secret key
    - APP__ACCESS_TOKEN_EXPIRE_MINUTES=30 # Access token expiration time
    - DATABASE__ENGINE=mysql # Database engine
    - DATABASE__HOST=db # Database host
    - DATABASE__USERNAME=user # Database username
    - DATABASE__PASSWORD=password # Database password
    - DATABASE__DB=ollama_hack # Database name
```

## 👤 Author

[Timlzh](https://github.com/timlzh)

## 📜 License

MIT License

## 🖼️ Screenshots

-   Home
    ![Home](./assets/index.png)
-   Endpoint Management
    ![Endpoint Management](./assets/endpoints.png)
-   Model Management
    ![Model Management](./assets/models.png)
-   Model Details
    ![Model Details](./assets/model_details.png)
-   Endpoint Details
    ![Endpoint Details](./assets/endpoint_details.png)
