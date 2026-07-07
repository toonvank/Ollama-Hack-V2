import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import axios from 'axios';
import { ApiClient, buildQueryString } from './client';

// Mock axios
vi.mock('axios', () => {
  const mockAxiosInstance = {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
    interceptors: {
      request: {
        use: vi.fn(),
      },
      response: {
        use: vi.fn(),
      },
    },
  };

  const mockAxios = vi.fn(() => mockAxiosInstance);
  mockAxios.create = vi.fn(() => mockAxiosInstance);
  mockAxios.interceptors = mockAxiosInstance.interceptors;

  return {
    default: mockAxios,
    __esModule: true,
  };
});

describe('buildQueryString', () => {
  it('should build query string from params', () => {
    const params = { page: 1, size: 10, search: 'test' };
    const result = buildQueryString(params);
    expect(result).toBe('?page=1&size=10&search=test');
  });

  it('should filter out undefined values', () => {
    const params = { page: 1, size: undefined, search: 'test' };
    const result = buildQueryString(params);
    expect(result).toBe('?page=1&search=test');
  });

  it('should filter out null values', () => {
    const params = { page: 1, size: null, search: 'test' };
    const result = buildQueryString(params);
    expect(result).toBe('?page=1&search=test');
  });

  it('should filter out empty string values', () => {
    const params = { page: 1, size: '', search: 'test' };
    const result = buildQueryString(params);
    expect(result).toBe('?page=1&search=test');
  });

  it('should return empty string for no params', () => {
    const params = {};
    const result = buildQueryString(params);
    expect(result).toBe('');
  });

  it('should return empty string when all values are filtered', () => {
    const params = { page: undefined, size: null, search: '' };
    const result = buildQueryString(params);
    expect(result).toBe('');
  });

  it('should encode special characters', () => {
    const params = { search: 'hello world', filter: 'a&b=c' };
    const result = buildQueryString(params);
    expect(result).toBe('?search=hello%20world&filter=a%26b%3Dc');
  });
});

describe('ApiClient', () => {
  let client: ApiClient;
  let mockGet: ReturnType<typeof vi.fn>;
  let mockPost: ReturnType<typeof vi.fn>;
  let mockPut: ReturnType<typeof vi.fn>;
  let mockPatch: ReturnType<typeof vi.fn>;
  let mockDelete: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.clearAllMocks();
    client = new ApiClient('http://test-api:8080');

    // Get the actual mocked methods from axios.create calls
    const axiosCreate = (axios.create as ReturnType<typeof vi.fn>).mock.results[0].value;
    mockGet = axiosCreate.get;
    mockPost = axiosCreate.post;
    mockPut = axiosCreate.put;
    mockPatch = axiosCreate.patch;
    mockDelete = axiosCreate.delete;
  });

  afterEach(() => {
    localStorage.clear();
  });

  describe('constructor', () => {
    it('should create client with default baseURL', () => {
      const defaultClient = new ApiClient();
      expect(defaultClient).toBeDefined();
    });

    it('should create client with custom baseURL', () => {
      const customClient = new ApiClient('http://custom:9000');
      expect(customClient).toBeDefined();
    });
  });

  describe('request interceptors', () => {
    it('should add auth token to requests when token exists', () => {
      localStorage.setItem('auth_token', 'test-token-123');
      new ApiClient('http://test:8080');

      const requestInterceptor = (axios.interceptors.request.use as ReturnType<typeof vi.fn>).mock.calls[0][0];
      const config = { headers: {} };
      const result = requestInterceptor(config);

      expect(result.headers.Authorization).toBe('Bearer test-token-123');
    });

    it('should not add auth token when token does not exist', () => {
      new ApiClient('http://test:8080');

      const requestInterceptor = (axios.interceptors.request.use as ReturnType<typeof vi.fn>).mock.calls[0][0];
      const config = { headers: {} };
      const result = requestInterceptor(config);

      expect(result.headers.Authorization).toBeUndefined();
    });
  });

  describe('GET request', () => {
    it('should make GET request and return data', async () => {
      const responseData = { items: [1, 2, 3] };
      mockGet.mockResolvedValue({ data: responseData });

      const result = await client.get('/test');

      expect(mockGet).toHaveBeenCalledWith('/test', undefined);
      expect(result).toEqual(responseData);
    });

    it('should make GET request with config', async () => {
      const config = { params: { page: 1 } };
      mockGet.mockResolvedValue({ data: {} });

      await client.get('/test', config);

      expect(mockGet).toHaveBeenCalledWith('/test', config);
    });
  });

  describe('POST request', () => {
    it('should make POST request and return data', async () => {
      const responseData = { id: 1, name: 'test' };
      mockPost.mockResolvedValue({ data: responseData });

      const result = await client.post('/test', { name: 'test' });

      expect(mockPost).toHaveBeenCalledWith('/test', { name: 'test' }, undefined);
      expect(result).toEqual(responseData);
    });

    it('should make POST request with config', async () => {
      mockPost.mockResolvedValue({ data: {} });

      await client.post('/test', { data: 'test' }, { headers: { 'X-Custom': 'header' } });

      expect(mockPost).toHaveBeenCalledWith('/test', { data: 'test' }, { headers: { 'X-Custom': 'header' } });
    });
  });

  describe('PUT request', () => {
    it('should make PUT request and return data', async () => {
      const responseData = { id: 1, updated: true };
      mockPut.mockResolvedValue({ data: responseData });

      const result = await client.put('/test/1', { name: 'updated' });

      expect(mockPut).toHaveBeenCalledWith('/test/1', { name: 'updated' }, undefined);
      expect(result).toEqual(responseData);
    });
  });

  describe('PATCH request', () => {
    it('should make PATCH request and return data', async () => {
      const responseData = { id: 1, patched: true };
      mockPatch.mockResolvedValue({ data: responseData });

      const result = await client.patch('/test/1', { status: 'active' });

      expect(mockPatch).toHaveBeenCalledWith('/test/1', { status: 'active' }, undefined);
      expect(result).toEqual(responseData);
    });
  });

  describe('DELETE request', () => {
    it('should make DELETE request and return data', async () => {
      const responseData = { success: true };
      mockDelete.mockResolvedValue({ data: responseData });

      const result = await client.delete('/test/1');

      expect(mockDelete).toHaveBeenCalledWith('/test/1', undefined);
      expect(result).toEqual(responseData);
    });

    it('should make DELETE request with config', async () => {
      mockDelete.mockResolvedValue({ data: {} });

      await client.delete('/test/1', { params: { force: true } });

      expect(mockDelete).toHaveBeenCalledWith('/test/1', { params: { force: true } });
    });
  });
});
