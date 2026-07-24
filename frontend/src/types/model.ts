import { AIModelStatusEnum } from "./common";

// AI Models Performance Information
export interface AIModelPerformance {
  id: number;
  status: AIModelStatusEnum;
  token_per_second?: number;
  connection_time?: number;
  total_time?: number;
  created_at: string;
}

// Model Info with Endpoint Info
export interface ModelFromEndpointInfo {
  id: number;
  url: string;
  name: string;
  created_at: string;
  endpoint_type?: string;
  /** Effective routing status (host + model + health) */
  status: AIModelStatusEnum;
  /** Raw endpoints.status */
  host_status?: string;
  /** Raw endpoint_ai_models.status (last model probe) */
  model_on_host?: string;
  token_per_second?: number;
  max_connection_time?: number;
  model_performances: AIModelPerformance[];
}

export interface AIModelInfoWithEndpointCount {
  id?: number;
  name: string;
  tag: string;
  enabled: boolean;
  created_at: string;
  endpoints: number;
  sglang_endpoints?: number;
  token_per_second?: number;
  max_connection_time?: number;
  param_billions?: number;
  composite_score?: number;
}

// AI Models details with Endpoints
export interface AIModelInfoWithEndpoint {
  id?: number;
  name: string;
  tag: string;
  created_at: string;
  total_endpoint_count: number;
  avaliable_endpoint_count: number;
  endpoints: {
    items: ModelFromEndpointInfo[];
    total?: number;
    page?: number;
    size?: number;
    pages?: number;
  };
}
