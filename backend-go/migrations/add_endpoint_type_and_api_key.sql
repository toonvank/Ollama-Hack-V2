-- Migration: Add endpoint_type and api_key to endpoints table
-- This allows supporting both Ollama native endpoints and OpenAI-compatible endpoints

-- Add endpoint_type column (default to 'ollama' for backward compatibility)
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS endpoint_type VARCHAR(50) DEFAULT 'ollama';

-- Add api_key column (nullable, only needed for authenticated endpoints)
ALTER TABLE endpoints ADD COLUMN IF NOT EXISTS api_key TEXT;

-- Update existing endpoints to explicitly set type as 'ollama'
UPDATE endpoints SET endpoint_type = 'ollama' WHERE endpoint_type IS NULL;

-- Add comment for documentation
COMMENT ON COLUMN endpoints.endpoint_type IS 'Type of endpoint: ollama (native Ollama API) or openai (OpenAI-compatible API)';
COMMENT ON COLUMN endpoints.api_key IS 'Optional API key for authenticated endpoints (e.g., Bearer token for OpenAI-compatible APIs)';
