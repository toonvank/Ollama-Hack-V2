import apiClient, { buildQueryString } from "./client";

import {
  BatchOperationResult,
  EndpointBatchCreate,
  EndpointBatchOperation,
  EndpointCreate,
  EndpointInfo,
  EndpointTaskInfo,
  EndpointUpdate,
  EndpointWithAIModelCount,
  EndpointWithAIModels,
  PageResponse,
  QueryParams,
} from "@/types";

export const endpointApi = {
  // Get all endpoints (with recent performance tests and AI model counts)
  getEndpoints: (params: QueryParams = {}) => {
    const queryString = buildQueryString({
      page: params.page || 1,
      size: params.size || 50,
      search: params.search,
      order_by: params.order_by,
      order: params.order,
      status: params.status,
    });

    return apiClient.get<PageResponse<EndpointWithAIModelCount>>(
      `/api/v2/endpoint${queryString}`,
    );
  },

  // Create New Endpoint
  createEndpoint: (data: EndpointCreate) => {
    return apiClient.post<EndpointInfo>("/api/v2/endpoint", data);
  },

  // Get single endpoint details (with AI models)
  getEndpointById: (
    endpointId: number,
    page: number = 1,
    size: number = 50,
  ) => {
    return apiClient.get<EndpointWithAIModels>(
      `/api/v2/endpoint/${endpointId}?page=${page}&size=${size}`,
    );
  },

  // UpdateEndpoints
  updateEndpoint: (endpointId: number, data: EndpointUpdate) => {
    return apiClient.patch<EndpointInfo>(
      `/api/v2/endpoint/${endpointId}`,
      data,
    );
  },

  // Delete Endpoint
  deleteEndpoint: (endpointId: number) => {
    return apiClient.delete<void>(`/api/v2/endpoint/${endpointId}`);
  },

  // Batch CreateEndpoints
  batchCreateEndpoints: (data: EndpointBatchCreate) => {
    return apiClient.post<EndpointInfo[]>("/api/v2/endpoint/batch", data);
  },

  // Batch test endpoints
  batchTestEndpoints: (data: EndpointBatchOperation) => {
    return apiClient.post<BatchOperationResult>(
      "/api/v2/endpoint/batch-test",
      data,
    );
  },

  // Batch delete endpoints
  batchDeleteEndpoints: (data: EndpointBatchOperation) => {
    return apiClient.delete<BatchOperationResult>("/api/v2/endpoint/batch", {
      data,
    });
  },

  // Manually trigger endpoint test
  triggerEndpointTest: (endpointId: number) => {
    return apiClient.post<void>(`/api/v2/endpoint/${endpointId}/test`);
  },

  // Get endpoint test results
  getEndpointTask: (endpointId: number) => {
    return apiClient.get<EndpointTaskInfo>(
      `/api/v2/endpoint/${endpointId}/task`,
    );
  },
};

export default endpointApi;
