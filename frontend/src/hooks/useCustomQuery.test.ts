import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@testing-library/react-query';
import { useCustomQuery } from './useCustomQuery';
import { ApiStatus } from '@/types';

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('useCustomQuery', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should fetch data successfully', async () => {
    const testData = { items: [1, 2, 3], total: 3 };
    const queryFn = vi.fn().mockResolvedValue(testData);

    const { result } = renderHook(
      () => useCustomQuery(['test-key'], queryFn),
      { wrapper: createWrapper() }
    );

    // Initially loading
    expect(result.current.isLoading).toBe(true);

    // Wait for data
    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data).toEqual(testData);
    expect(queryFn).toHaveBeenCalledTimes(1);
  });

  it('should handle error state', async () => {
    const error = { status: 500, message: 'Server error' };
    const queryFn = vi.fn().mockRejectedValue(error);

    const { result } = renderHook(
      () => useCustomQuery(['test-key'], queryFn),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });

    expect(result.current.error).toEqual(error);
  });

  it('should use provided query options', async () => {
    const testData = { value: 'test' };
    const queryFn = vi.fn().mockResolvedValue(testData);

    const { result } = renderHook(
      () => useCustomQuery(['test-key'], queryFn, {
        enabled: false,
      }),
      { wrapper: createWrapper() }
    );

    // Should not fetch when enabled is false
    expect(queryFn).not.toHaveBeenCalled();
    expect(result.current.isPending).toBe(true);
  });

  it('should refetch when refetch is called', async () => {
    const testData = { count: 1 };
    const queryFn = vi.fn().mockResolvedValue(testData);

    const { result } = renderHook(
      () => useCustomQuery(['test-key'], queryFn),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(queryFn).toHaveBeenCalledTimes(1);

    // Trigger refetch
    result.current.refetch();

    await waitFor(() => {
      expect(queryFn).toHaveBeenCalledTimes(2);
    });
  });

  it('should return proper query state', async () => {
    const testData = { data: 'test' };
    const queryFn = vi.fn().mockResolvedValue(testData);

    const { result } = renderHook(
      () => useCustomQuery(['test-key', 'param1'], queryFn),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.queryKey).toEqual(['test-key', 'param1']);
    expect(result.current.data).toEqual(testData);
    expect(result.current.isLoading).toBe(false);
    expect(result.current.isFetching).toBe(false);
  });
});
