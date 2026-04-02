import axios, {
  AxiosError,
  AxiosInstance,
  AxiosRequestConfig,
  AxiosResponse,
} from "axios";

// Define base API response structure
export interface ApiResponse<T = any> {
  data: T;
  success: boolean;
  message?: string;
}

// Extend AxiosError interface to include detail field
export interface EnhancedAxiosError extends AxiosError {
  detail?: string;
}

// Convert query params object to URL query string
export const buildQueryString = (params: Record<string, any>): string => {
  const query = Object.entries(params)
    .filter(
      ([_, value]) => value !== undefined && value !== null && value !== "",
    )
    .map(
      ([key, value]) =>
        `${encodeURIComponent(key)}=${encodeURIComponent(value)}`,
    )
    .join("&");

  return query ? `?${query}` : "";
};

export class ApiClient {
  private client: AxiosInstance;
  private baseURL: string;

  constructor(baseURL: string = "http://localhost:8000") {
    this.baseURL = baseURL;
    this.client = axios.create({
      baseURL,
      headers: {
        "Content-Type": "application/json",
      },
    });

    this.setupInterceptors();
  }

  private setupInterceptors() {
    // Request interceptor - add auth token
    this.client.interceptors.request.use(
      (config) => {
        const token = localStorage.getItem("auth_token");

        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }

        return config;
      },
      (error) => Promise.reject(error),
    );

    // Response interceptor - handle errors and refresh token
    this.client.interceptors.response.use(
      (response) => response,
      async (error: AxiosError) => {
        const originalRequest = error.config as AxiosRequestConfig & {
          _retry?: boolean;
        };

        // Handle detail field in error response
        if (error.response?.data) {
          const responseData = error.response.data;

          // Check if response contains detail field
          if (typeof responseData === "object" && "detail" in responseData) {
            const detail = responseData.detail;

            if (typeof detail === "string") {
              (error as EnhancedAxiosError).detail = detail;
            }
          }
        }

        // Handle 401 error (unauthorized)
        if (
          error.response?.status === 401 &&
          !originalRequest._retry &&
          window.location.pathname !== "/init" &&
          window.location.pathname !== "/login"
        ) {
          // If token refresh is needed, add logic here
          // Current simple implementation: clear token on 401 and redirect to login
          localStorage.removeItem("auth_token");
          window.location.href = "/login";

          return Promise.reject(error);
        }

        return Promise.reject(error);
      },
    );
  }

  // Generic GET request
  public async get<T = any>(
    url: string,
    config?: AxiosRequestConfig,
  ): Promise<T> {
    const response: AxiosResponse = await this.client.get(url, config);

    return response.data;
  }

  // Generic POST request
  public async post<T = any>(
    url: string,
    data?: any,
    config?: AxiosRequestConfig,
  ): Promise<T> {
    const response: AxiosResponse = await this.client.post(url, data, config);

    return response.data;
  }

  // Generic PUT request
  public async put<T = any>(
    url: string,
    data?: any,
    config?: AxiosRequestConfig,
  ): Promise<T> {
    const response: AxiosResponse = await this.client.put(url, data, config);

    return response.data;
  }

  // Generic PATCH request
  public async patch<T = any>(
    url: string,
    data?: any,
    config?: AxiosRequestConfig,
  ): Promise<T> {
    const response: AxiosResponse = await this.client.patch(url, data, config);

    return response.data;
  }

  // Generic DELETE request
  public async delete<T = any>(
    url: string,
    config?: AxiosRequestConfig,
  ): Promise<T> {
    const response: AxiosResponse = await this.client.delete(url, config);

    return response.data;
  }
}
const baseURL = import.meta.env.VITE_API_BASE_URL;

// Create default client instance
export const apiClient = new ApiClient(baseURL || "http://localhost:8000");

export default apiClient;
