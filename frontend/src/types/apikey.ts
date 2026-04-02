// API Key creation request
export interface ApiKeyCreate {
  name: string;
}

// API Key info
export interface ApiKeyInfo {
  id: number;
  name: string;
  created_at: string;
  last_used_at?: string;
  user_id?: number;
  user_name?: string;
}

// API Key response (includes key value)
export interface ApiKeyResponse extends ApiKeyInfo {
  key: string;
}

// API KeysUsage Statistics
export interface ApiKeyUsageStats {
  total_requests: number;
  last_30_days_requests: number;
  requests_today: number;
  successful_requests: number;
  failed_requests: number;
  requests_per_day: {
    date: string;
    count: number;
  }[];
}
