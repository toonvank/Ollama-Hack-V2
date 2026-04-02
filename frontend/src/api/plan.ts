import apiClient, { buildQueryString } from "./client";

import {
  PageResponse,
  PlanCreate,
  PlanResponse,
  PlanUpdate,
  QueryParams,
} from "@/types";

export const planApi = {
  // Get all plans
  getPlans: (params: QueryParams = {}) => {
    const queryString = buildQueryString({
      page: params.page || 1,
      size: params.size || 50,
      search: params.search,
      order_by: params.order_by,
      order: params.order,
    });

    return apiClient.get<PageResponse<PlanResponse>>(
      `/api/v2/plan/${queryString}`,
    );
  },

  // Create New Plan
  createPlan: (data: PlanCreate) => {
    return apiClient.post<PlanResponse>("/api/v2/plan/", data);
  },

  // Get plan by ID
  getPlanById: (planId: number) => {
    return apiClient.get<PlanResponse>(`/api/v2/plan/${planId}`);
  },

  // Get current user plan
  getCurrentUserPlan: () => {
    return apiClient.get<PlanResponse>("/api/v2/plan/me");
  },

  // Update plan
  updatePlan: (planId: number, data: PlanUpdate) => {
    return apiClient.patch<PlanResponse>(`/api/v2/plan/${planId}`, data);
  },

  // Delete plan
  deletePlan: (planId: number) => {
    return apiClient.delete<void>(`/api/v2/plan/${planId}`);
  },
};

export default planApi;
