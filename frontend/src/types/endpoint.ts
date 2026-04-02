import {
  AIModelStatusEnum,
  EndpointStatusEnum,
  TaskStatusEnum,
} from "./common";

// Endpoint Info
export interface EndpointInfo {
  id?: number;
  url: string;
  name: string;
  created_at?: string;
}

// Endpoint Performance Information
export interface EndpointPerformanceInfo {
  id?: number;
  status: EndpointStatusEnum;
  ollama_version?: string;
  created_at: string;
}

// Endpoint Info with AI Models count
export interface EndpointWithAIModelCount extends EndpointInfo {
  recent_performances: EndpointPerformanceInfo[];
  total_ai_model_count: number;
  avaliable_ai_model_count: number;
  task_status?: TaskStatusEnum;
}

// Endpoint AI Models Information
export interface EndpointAIModelInfo {
  id: number;
  name: string;
  tag: string;
  created_at: string;
  status: AIModelStatusEnum;
  token_per_second?: number;
  max_connection_time?: number;
}

// Endpoint Info with AI Models list
export interface EndpointWithAIModels extends EndpointInfo {
  recent_performances: EndpointPerformanceInfo[];
  ai_models: {
    items: EndpointAIModelInfo[];
    total?: number;
    page?: number;
    size?: number;
    pages?: number;
  };
}

// Endpoint Create Request
export interface EndpointCreate {
  url: string;
  name?: string;
}

// Endpoint Update Request
export interface EndpointUpdate {
  name?: string;
}

// Batch Create Endpoints Request
export interface EndpointBatchCreate {
  endpoints: EndpointCreate[];
}

// Batch Action Endpoints Request
export interface EndpointBatchOperation {
  endpoint_ids: number[]; // List of endpoint IDs to perform action on
}

// Batch Action Results Response
export interface BatchOperationResult {
  success_count: number; // Number of successfully processed endpoints
  failed_count: number; // Number of failed endpoints
  failed_ids: {
    // Failed endpoint IDs and reasons
    [key: string]: string;
  };
}

// Endpoint Task Information
export interface EndpointTaskInfo {
  id: number;
  endpoint_id: number;
  status: TaskStatusEnum;
  scheduled_at: string;
  last_tried?: string;
  created_at: string;
}
