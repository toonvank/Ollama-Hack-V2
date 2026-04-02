import apiClient, { buildQueryString } from "./client";

import {
  AIModelInfoWithEndpoint,
  AIModelInfoWithEndpointCount,
  PageResponse,
  QueryParams,
} from "@/types";

export const aiModelApi = {
  // Get all AI models (with recent performance tests)
  getAIModels: (params: QueryParams = {}) => {
    const queryString = buildQueryString({
      page: params.page || 1,
      size: params.size || 50,
      search: params.search,
      order_by: params.order_by,
      order: params.order,
      is_available: params.is_available,
    });

    return apiClient.get<PageResponse<AIModelInfoWithEndpointCount>>(
      `/api/v2/ai_model/${queryString}`,
    );
  },

  // Get single AI model details (with endpoints)
  getAIModelById: (modelId: number, page: number = 1, size: number = 50) => {
    return apiClient.get<AIModelInfoWithEndpoint>(
      `/api/v2/ai_model/${modelId}?page=${page}&size=${size}`,
    );
  },
};

export default aiModelApi;
