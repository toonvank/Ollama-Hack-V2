import apiClient, { buildQueryString } from "./client";

import {
  ChangePasswordRequest,
  PageResponse,
  QueryParams,
  Token,
  UserAuth,
  UserInfo,
  UserUpdate,
} from "@/types";

export const authApi = {
  // Initialize first user
  initUser: (data: UserAuth) => {
    return apiClient.post<void>("/api/v2/user/init", data);
  },

  // UserSign In
  login: (username: string, password: string) => {
    return apiClient.post<Token>("/api/v2/user/login", { username, password });
  },

  // Get current user info
  getCurrentUser: () => {
    return apiClient.get<UserInfo>("/api/v2/user/me");
  },

  // Change current user password
  changePassword: (data: ChangePasswordRequest) => {
    return apiClient.patch<UserInfo>(
      `/api/v2/user/me/change-password?old_password=${data.old_password}&new_password=${data.new_password}`,
    );
  },

  // Create new user (requires admin privileges)
  createUser: (data: UserAuth) => {
    return apiClient.post<UserInfo>("/api/v2/user", data);
  },

  // Get all users (requires admin privileges)
  getUsers: (params: QueryParams = {}) => {
    const queryString = buildQueryString({
      page: params.page || 1,
      size: params.size || 50,
      search: params.search,
      order_by: params.order_by,
      order: params.order,
    });

    return apiClient.get<PageResponse<UserInfo>>(`/api/v2/user${queryString}`);
  },

  // Get user by ID (requires admin privileges)
  getUserById: (userId: number) => {
    return apiClient.get<UserInfo>(`/api/v2/user/${userId}`);
  },

  // Update user (requires admin privileges)
  updateUser: (userId: number, data: UserUpdate) => {
    return apiClient.patch<UserInfo>(`/api/v2/user/${userId}`, data);
  },

  // Delete user (requires admin privileges)
  deleteUser: (userId: number) => {
    return apiClient.delete<void>(`/api/v2/user/${userId}`);
  },
};

export default authApi;
