-- Migration: Per-API-key model filter
-- Adds allowed_models to api_keys: comma-separated model ids (NULL = unrestricted).
-- Enforced on /v1/*, /api/* proxying and the /v1/models + /api/tags listings.
--
-- Match semantics (case-insensitive):
--   "name:tag"  -> exact id match (e.g. "glm-4.7-flash:latest")
--   "name"      -> matches the id and every tag variant of that model
--   "prefix*"   -> wildcard prefix match (e.g. "glm-5.3*")
-- Pseudo-models (smart:*) resolve within the allowed pool automatically.

ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS allowed_models TEXT;

COMMENT ON COLUMN api_keys.allowed_models IS 'Comma-separated model filter; NULL/empty = all models allowed';
