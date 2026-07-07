import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useCustomMutation } from './useCustomMutation';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      mutations: {
        retry: false,
      },
    },
  });

  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('useCustomMutation', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should execute mutation successfully', async () => {
    const mutationFn = vi.fn().mockResolvedValue({ id: 1, name: 'created' });

    const { result } = renderHook(
      () => useCustomMutation(mutationFn),
      { wrapper: createWrapper() }
    );

    await act(async () => {
      await result.current.mutateAsync({ name: 'test' });
    });

    expect(mutationFn).toHaveBeenCalledWith({ name: 'test' });
    expect(result.current.data).toEqual({ id: 1, name: 'created' });
    expect(result.current.isSuccess).toBe(true);
  });

  it('should handle mutation error', async () => {
    const error = { status: 400, message: 'Bad request' };
    const mutationFn = vi.fn().mockRejectedValue(error);

    const { result } = renderHook(
      () => useCustomMutation(mutationFn),
      { wrapper: createWrapper() }
    );

    await act(async () => {
      try {
        await result.current.mutateAsync({ name: 'test' });
      } catch {
        // Expected error
      }
    });

    expect(result.current.isError).toBe(true);
    expect(result.current.error).toEqual(error);
  });

  it('should show loading state during mutation', async () => {
    let resolveMutation: (value: unknown) => void;
    const promise = new Promise((resolve) => {
      resolveMutation = resolve;
    });

    const mutationFn = vi.fn().mockImplementation(() => promise);

    const { result } = renderHook(
      () => useCustomMutation(mutationFn),
      { wrapper: createWrapper() }
    );

    // Start mutation but don't await it
    const mutationPromise = act(async () => {
      return result.current.mutateAsync({ name: 'test' });
    });

    // Check loading state before resolution
    expect(result.current.isPending).toBe(true);

    // Resolve the mutation
    resolveMutation!({ id: 1 });
    await mutationPromise;
  });

  it('should call onSuccess callback', async () => {
    const onSuccess = vi.fn();
    const mutationFn = vi.fn().mockResolvedValue({ id: 1 });

    const { result } = renderHook(
      () => useCustomMutation(mutationFn, { onSuccess }),
      { wrapper: createWrapper() }
    );

    await act(async () => {
      await result.current.mutateAsync({ name: 'test' });
    });

    expect(onSuccess).toHaveBeenCalledWith({ id: 1 }, { name: 'test' }, undefined);
  });

  it('should call onError callback on failure', async () => {
    const onError = vi.fn();
    const error = { status: 500, message: 'Server error' };
    const mutationFn = vi.fn().mockRejectedValue(error);

    const { result } = renderHook(
      () => useCustomMutation(mutationFn, { onError }),
      { wrapper: createWrapper() }
    );

    await act(async () => {
      try {
        await result.current.mutateAsync({ name: 'test' });
      } catch {
        // Expected
      }
    });

    expect(onError).toHaveBeenCalledWith(error, { name: 'test' }, undefined);
  });

  it('should call onSettled callback regardless of outcome', async () => {
    const onSettled = vi.fn();
    const mutationFn = vi.fn().mockResolvedValue({ id: 1 });

    const { result } = renderHook(
      () => useCustomMutation(mutationFn, { onSettled }),
      { wrapper: createWrapper() }
    );

    await act(async () => {
      await result.current.mutateAsync({ name: 'test' });
    });

    expect(onSettled).toHaveBeenCalled();
  });

  it('should reset mutation state with reset function', async () => {
    const mutationFn = vi.fn().mockResolvedValue({ id: 1 });

    const { result } = renderHook(
      () => useCustomMutation(mutationFn),
      { wrapper: createWrapper() }
    );

    await act(async () => {
      await result.current.mutateAsync({ name: 'test' });
    });

    expect(result.current.data).toBeDefined();

    act(() => {
      result.current.reset();
    });

    expect(result.current.data).toBeUndefined();
  });

  it('should use provided mutation options', async () => {
    const mutationFn = vi.fn().mockResolvedValue({ id: 1 });

    const { result } = renderHook(
      () => useCustomMutation(mutationFn, {
        mutationKey: ['test-mutation'],
      }),
      { wrapper: createWrapper() }
    );

    await act(async () => {
      await result.current.mutateAsync({ name: 'test' });
    });

    expect(result.current.mutationKey).toEqual(['test-mutation']);
  });

  it('should support variables with void return', async () => {
    const mutationFn = vi.fn().mockResolvedValue(undefined);

    const { result } = renderHook(
      () => useCustomMutation<void, { id: number }>(mutationFn),
      { wrapper: createWrapper() }
    );

    await act(async () => {
      await result.current.mutateAsync({ id: 1 });
    });

    expect(mutationFn).toHaveBeenCalledWith({ id: 1 });
  });

  it('should track mutation variables', async () => {
    const mutationFn = vi.fn().mockResolvedValue({ result: 'ok' });

    const { result } = renderHook(
      () => useCustomMutation(mutationFn),
      { wrapper: createWrapper() }
    );

    const variables = { name: 'test', value: 42 };

    await act(async () => {
      await result.current.mutateAsync(variables);
    });

    expect(result.current.variables).toEqual(variables);
  });
});
