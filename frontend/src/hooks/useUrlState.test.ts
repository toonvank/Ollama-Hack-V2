import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { MemoryRouter, useSearchParams } from 'react-router-dom';
import { useUrlState, usePaginationUrlState } from './useUrlState';

const createWrapper = () => {
  return ({ children }: { children: React.ReactNode }) => (
    <MemoryRouter>{children}</MemoryRouter>
  );
};

describe('useUrlState', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should use initial state when no URL param exists', () => {
    const initialState = { filter: 'all' };

    const { result } = renderHook(
      () => useUrlState(initialState, { paramName: 'filter' }),
      { wrapper: createWrapper() }
    );

    const [state] = result.current;
    expect(state).toEqual({ filter: 'all' });
  });

  it('should read state from URL param on mount', () => {
    const TestComponent = () => {
      const [searchParams] = useSearchParams();
      searchParams.set('filter', 'active');

      const initialState = { filter: 'all' };
      const result = useUrlState(initialState, { paramName: 'filter' });

      return { state: result[0] };
    };

    const { result } = renderHook(() => TestComponent(), {
      wrapper: createWrapper()
    });

    expect(result.current.state).toEqual({ filter: 'active' });
  });

  it('should update URL when state changes', () => {
    const initialState = { count: 0 };

    const { result } = renderHook(
      () => useUrlState(initialState, { paramName: 'count' }),
      { wrapper: createWrapper() }
    );

    const [, updateState] = result.current;

    act(() => {
      updateState({ count: 5 });
    });

    const [state] = result.current;
    expect(state.count).toBe(5);
  });

  it('should use custom serialize/deserialize functions', () => {
    const initialState = { value: 10 };

    const { result } = renderHook(
      () => useUrlState(initialState, {
        paramName: 'data',
        serialize: (val) => `prefix-${val.value}`,
        deserialize: (str) => ({ value: parseInt(str.replace('prefix-', '')) })
      }),
      { wrapper: createWrapper() }
    );

    const [state] = result.current;
    expect(state).toEqual({ value: 10 });
  });

  it('should fall back to initial state on deserialize error', () => {
    const initialState = { data: 'default' };

    const TestComponent = () => {
      const [searchParams] = useSearchParams();
      searchParams.set('data', 'invalid-json{{{');

      const result = useUrlState(initialState, {
        paramName: 'data',
        deserialize: JSON.parse
      });

      return { state: result[0] };
    };

    const { result } = renderHook(() => TestComponent(), {
      wrapper: createWrapper()
    });

    expect(result.current.state).toEqual({ data: 'default' });
  });

  it('should use replace state when option is set', () => {
    const initialState = { value: 1 };

    const { result } = renderHook(
      () => useUrlState(initialState, {
        paramName: 'value',
        replaceState: true
      }),
      { wrapper: createWrapper() }
    );

    const [, updateState] = result.current;

    act(() => {
      updateState({ value: 2 });
    });

    expect(result.current[0].value).toBe(2);
  });

  it('should handle functional state updates', () => {
    const initialState = { count: 0 };

    const { result } = renderHook(
      () => useUrlState(initialState, { paramName: 'count' }),
      { wrapper: createWrapper() }
    );

    const [, updateState] = result.current;

    act(() => {
      updateState((prev) => ({ count: prev.count + 1 }));
    });

    const [state] = result.current;
    expect(state.count).toBe(1);
  });
});

describe('usePaginationUrlState', () => {
  it('should use default initial state', () => {
    const { result } = renderHook(
      () => usePaginationUrlState({
        page: 1,
        pageSize: 10,
        search: '',
        orderBy: 'name',
        order: 'asc'
      }),
      { wrapper: createWrapper() }
    );

    expect(result.current.page).toBe(1);
    expect(result.current.pageSize).toBe(10);
  });

  it('should validate minimum page number', () => {
    const { result } = renderHook(
      () => usePaginationUrlState(
        { page: 0, pageSize: 10 },
        { page: { min: 1 } }
      ),
      { wrapper: createWrapper() }
    );

    expect(result.current.page).toBe(1);
  });

  it('should validate maximum page number', () => {
    const { result } = renderHook(
      () => usePaginationUrlState(
        { page: 100, pageSize: 10 },
        { page: { max: 50 } }
      ),
      { wrapper: createWrapper() }
    );

    expect(result.current.page).toBe(50);
  });

  it('should respect totalPages constraint', () => {
    const { result } = renderHook(
      () => usePaginationUrlState(
        { page: 10, pageSize: 10 },
        { page: { min: 1 }, totalPages: 5 }
      ),
      { wrapper: createWrapper() }
    );

    expect(result.current.page).toBe(5);
  });

  it('should validate pageSize within range', () => {
    const { result } = renderHook(
      () => usePaginationUrlState(
        { page: 1, pageSize: 5 },
        { pageSize: { min: 10, max: 100 } }
      ),
      { wrapper: createWrapper() }
    );

    expect(result.current.pageSize).toBe(10);
  });

  it('should reset page when changing pageSize', () => {
    const { result } = renderHook(
      () => usePaginationUrlState({ page: 5, pageSize: 10 }),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.setPageSize(20);
    });

    expect(result.current.page).toBe(1);
    expect(result.current.pageSize).toBe(20);
  });

  it('should reset page when changing search', () => {
    const { result } = renderHook(
      () => usePaginationUrlState({ page: 5, search: '' }),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.setSearch('test');
    });

    expect(result.current.page).toBe(1);
    expect(result.current.search).toBe('test');
  });

  it('should validate orderBy against allowed fields', () => {
    const { result } = renderHook(
      () => usePaginationUrlState(
        { page: 1, orderBy: 'invalid', order: 'asc' },
        { orderBy: { allowedFields: ['name', 'date'], defaultField: 'name' } }
      ),
      { wrapper: createWrapper() }
    );

    expect(result.current.orderBy).toBe('name');
  });

  it('should provide setOrderBy method', () => {
    const { result } = renderHook(
      () => usePaginationUrlState({ page: 1, orderBy: 'name', order: 'asc' }),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.setOrderBy('date');
    });

    expect(result.current.orderBy).toBe('date');
  });

  it('should provide setOrder method', () => {
    const { result } = renderHook(
      () => usePaginationUrlState({ page: 1, order: 'asc' }),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.setOrder('desc');
    });

    expect(result.current.order).toBe('desc');
  });

  it('should provide setCustomParam method', () => {
    const { result } = renderHook(
      () => usePaginationUrlState({ page: 1, custom: 'value' }),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.setCustomParam('filter', 'active');
    });

    expect(result.current.filter).toBe('active');
  });

  it('should handle setState with validation', () => {
    const { result } = renderHook(
      () => usePaginationUrlState(
        { page: 1, pageSize: 10 },
        { pageSize: { min: 5, max: 50 } }
      ),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.setState({ page: 1, pageSize: 100 });
    });

    expect(result.current.pageSize).toBe(50);
  });
});
