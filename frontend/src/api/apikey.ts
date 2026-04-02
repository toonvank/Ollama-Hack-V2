import apiClient, { buildQueryString } from "./client";

import {
  ApiKeyCreate,
  ApiKeyInfo,
  ApiKeyResponse,
  ApiKeyUsageStats,
  PageResponse,
  QueryParams,
} from "@/types";

export const apiKeyApi = {
  // Get all API keys for current user
  getApiKeys: (params: QueryParams = {}) => {
    const queryString = buildQueryString({
      page: params.page || 1,
      size: params.size || 50,
      search: params.search,
      order_by: params.order_by,
      order: params.order,
    });

    return apiClient.get<PageResponse<ApiKeyInfo>>(
      `/api/v2/apikey/${queryString}`,
    );
  },

  // Create new API key
  createApiKey: (data: ApiKeyCreate) => {
    return apiClient.post<ApiKeyResponse>("/api/v2/apikey/", data);
  },

  // Delete API Keys
  deleteApiKey: (apiKeyId: number) => {
    return apiClient.delete<void>(`/api/v2/apikey/${apiKeyId}`);
  },

  // Get API key usage statistics
  getApiKeyStats: (apiKeyId: number) => {
    return apiClient.get<ApiKeyUsageStats>(`/api/v2/apikey/${apiKeyId}/stats`);
  },
};

export default apiKeyApi;
