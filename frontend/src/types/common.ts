// Paginated response generic type
export interface PageResponse<T> {
  items: T[];
  total?: number;
  page?: number;
  size?: number;
  pages?: number;
}

// API request generic parameters
export interface PaginationParams {
  page?: number;
  size?: number;
}

// Sort direction enum
export enum SortOrder {
  ASC = "asc",
  DESC = "desc",
}

// Generic query parameter interface
export interface QueryParams extends PaginationParams {
  search?: string;
  order_by?: string;
  order?: SortOrder;
}

// API error response
export interface ApiError {
  status: number;
  message: string;
  details?: unknown;
}

// API status enum
export enum ApiStatus {
  IDLE = "idle",
  LOADING = "loading",
  SUCCESS = "success",
  ERROR = "error",
}

// Generic status type
export type ApiState<T> = {
  data: T | null;
  status: ApiStatus;
  error: ApiError | null;
};

// Enum type
export enum AIModelStatusEnum {
  AVAILABLE = "available",
  UNAVAILABLE = "unavailable",
  FAKE = "fake",
  MISSING = "missing",
}

export enum EndpointStatusEnum {
  AVAILABLE = "available",
  UNAVAILABLE = "unavailable",
  FAKE = "fake",
}

// Task status enum
export enum TaskStatusEnum {
  PENDING = "pending",
  RUNNING = "running",
  DONE = "done",
  FAILED = "failed",
}
