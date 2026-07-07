import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { AuthProvider, useAuth } from './AuthContext';
import { authApi } from '@/api';

// Mock authApi
vi.mock('@/api', () => ({
  authApi: {
    login: vi.fn(),
    getCurrentUser: vi.fn(),
  },
  apiClient: {},
}));

const createWrapper = () => {
  return ({ children }: { children: React.ReactNode }) => (
    <AuthProvider>{children}</AuthProvider>
  );
};

describe('AuthProvider', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('should initialize with loading state', async () => {
    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    expect(result.current.isLoading).toBe(true);
  });

  it('should load user from token on mount', async () => {
    const mockUser = { id: 1, username: 'test', is_admin: false };
    (authApi.getCurrentUser as ReturnType<typeof vi.fn>).mockResolvedValue(mockUser);

    localStorage.setItem('auth_token', 'test-token');

    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.user).toEqual(mockUser);
    expect(result.current.token).toBe('test-token');
  });

  it('should handle missing token on mount', async () => {
    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.user).toBeNull();
    expect(result.current.token).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
  });

  it('should handle invalid token', async () => {
    (authApi.getCurrentUser as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Invalid token'));

    localStorage.setItem('auth_token', 'invalid-token');

    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.user).toBeNull();
    expect(result.current.token).toBeNull();
    expect(localStorage.getItem('auth_token')).toBeNull();
  });

  it('should login successfully', async () => {
    const mockResponse = { access_token: 'new-token' };
    const mockUser = { id: 1, username: 'test', is_admin: true };

    (authApi.login as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse);
    (authApi.getCurrentUser as ReturnType<typeof vi.fn>).mockResolvedValue(mockUser);

    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    await act(async () => {
      await result.current.login('testuser', 'password');
    });

    expect(localStorage.getItem('auth_token')).toBe('new-token');
    expect(result.current.token).toBe('new-token');
  });

  it('should handle login error', async () => {
    const loginError = new Error('Invalid credentials');
    (authApi.login as ReturnType<typeof vi.fn>).mockRejectedValue(loginError);

    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    await expect(result.current.login('testuser', 'wrong')).rejects.toThrow('Invalid credentials');
  });

  it('should logout', async () => {
    const mockUser = { id: 1, username: 'test', is_admin: false };
    (authApi.getCurrentUser as ReturnType<typeof vi.fn>).mockResolvedValue(mockUser);

    localStorage.setItem('auth_token', 'test-token');

    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    act(() => {
      result.current.logout();
    });

    expect(result.current.user).toBeNull();
    expect(result.current.token).toBeNull();
    expect(localStorage.getItem('auth_token')).toBeNull();
  });

  it('should return correct isAuthenticated state', async () => {
    const mockUser = { id: 1, username: 'test', is_admin: false };
    (authApi.getCurrentUser as ReturnType<typeof vi.fn>).mockResolvedValue(mockUser);

    localStorage.setItem('auth_token', 'test-token');

    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true);
    });

    act(() => {
      result.current.logout();
    });

    expect(result.current.isAuthenticated).toBe(false);
  });

  it('should return correct isAdmin state', async () => {
    const mockUser = { id: 1, username: 'test', is_admin: true };
    (authApi.getCurrentUser as ReturnType<typeof vi.fn>).mockResolvedValue(mockUser);

    localStorage.setItem('auth_token', 'test-token');

    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isAdmin).toBe(true);
    });
  });

  it('should provide getTokenFromStorage method', () => {
    localStorage.setItem('auth_token', 'stored-token');

    const { result } = renderHook(() => useAuth(), { wrapper: createWrapper() });

    expect(result.current.getTokenFromStorage()).toBe('stored-token');
  });

  it('should throw error when useAuth used outside AuthProvider', () => {
    // Suppress console.error for this test
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    expect(() => {
      renderHook(() => useAuth());
    }).toThrow('useAuth must be used within an AuthProvider');

    consoleSpy.mockRestore();
  });
});
