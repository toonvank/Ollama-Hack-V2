import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";

/**
 * Hook for URL parameter state management
 * @param param0 parameter configuration
 * @returns State and update functions
 */
export function useUrlState<T>(
  initialState: T,
  options: {
    paramName: string;
    serialize?: (value: T) => string;
    deserialize?: (value: string) => T;
    replaceState?: boolean;
  },
) {
  const {
    paramName,
    serialize = JSON.stringify,
    deserialize = JSON.parse,
    replaceState = false,
  } = options;

  const [searchParams, setSearchParams] = useSearchParams();

  // Get initial value from URL parameter, use initialState if not present
  const getInitialValue = useCallback(() => {
    const paramValue = searchParams.get(paramName);

    if (paramValue) {
      try {
        return deserialize(paramValue);
      } catch {
        return initialState;
      }
    }

    return initialState;
  }, [deserialize, initialState, paramName, searchParams]);

  const [state, setState] = useState<T>(getInitialValue);

  // Sync URL parameter when updating state
  const updateState = useCallback(
    (newState: T | ((prevState: T) => T)) => {
      setState((prevState) => {
        const nextState =
          typeof newState === "function"
            ? (newState as (prevState: T) => T)(prevState)
            : newState;

        try {
          const serialized = serialize(nextState);

          setSearchParams(
            (prev) => {
              const next = new URLSearchParams(prev);

              if (serialized === serialize(initialState)) {
                next.delete(paramName);
              } else {
                next.set(paramName, serialized);
              }

              return next;
            },
            { replace: replaceState },
          );
        } catch {
          // Ignore errors
        }

        return nextState;
      });
    },
    [initialState, paramName, replaceState, serialize, setSearchParams],
  );

  // Update state when URL parameter changes
  useEffect(() => {
    const paramValue = searchParams.get(paramName);

    if (paramValue) {
      try {
        const parsedValue = deserialize(paramValue);
        setState((current) => JSON.stringify(current) !== JSON.stringify(parsedValue) ? parsedValue : current);
      } catch {
        // Ignore errors
      }
    } else if (searchParams.toString() !== "") {
      // Reset to initial value only when other parameters exist but current parameter does not
      setState((current) => JSON.stringify(current) !== JSON.stringify(initialState) ? initialState : current);
    }
  }, [searchParams.get(paramName)]);

  return [state, updateState] as const;
}

/**
 * Pagination params validation configuration interface
 */
export interface PaginationValidationConfig {
  // Page number validation
  page?: {
    min?: number; // Minimum page number, default is1
    max?: number; // Maximum page number, automatically calculated if totalPages is provided
  };
  // Per page count validation
  pageSize?: {
    min?: number; // Minimum per page count
    max?: number; // Maximum per page count
  };
  // sort fieldvalidation
  orderBy?: {
    allowedFields?: string[]; // allowedofsortingfield list
    defaultField?: string; // Default sort field
  };
  // Total pages, for page fallback functionality
  totalPages?: number;
}

/**
 * More generic URL parameter state hook that can manage multiple parameters
 */
export function usePaginationUrlState<SortType = string>(
  initialState: {
    page?: number;
    pageSize?: number;
    search?: string;
    orderBy?: SortType;
    order?: string;
    [key: string]: any;
  },
  validationConfig?: PaginationValidationConfig,
) {
  const [searchParams, setSearchParams] = useSearchParams();

  // Get initial state from URL
  const getInitialStateFromUrl = () => {
    const stateFromUrl = { ...initialState };

    // Parse page number
    const page = searchParams.get("page");

    if (page && !isNaN(Number(page))) {
      stateFromUrl.page = Number(page);
    }

    // Parse per page count
    const pageSize = searchParams.get("pageSize");

    if (pageSize && !isNaN(Number(pageSize))) {
      stateFromUrl.pageSize = Number(pageSize);
    }

    // Parse search term
    const search = searchParams.get("search");

    if (search) {
      stateFromUrl.search = search;
    }

    // Parse sort field
    const orderBy = searchParams.get("orderBy");

    if (orderBy) {
      if (validationConfig?.orderBy?.allowedFields?.includes(orderBy)) {
        stateFromUrl.orderBy = orderBy as unknown as SortType;
      } else {
        stateFromUrl.orderBy = validationConfig?.orderBy?.defaultField as unknown as SortType;
      }
    }

    // Parse sorting direction
    const order = searchParams.get("order");

    if (order) {
      stateFromUrl.order = order;
    }

    // Handle other custom parameters
    Object.keys(initialState).forEach((key) => {
      if (!["page", "pageSize", "search", "orderBy", "order"].includes(key)) {
        const value = searchParams.get(key);

        if (value) {
          try {
            stateFromUrl[key] = JSON.parse(value);
          } catch {
            stateFromUrl[key] = value;
          }
        }
      }
    });

    // Apply validation rules
    if (validationConfig) {
      // Validate page number
      if (validationConfig.page && typeof stateFromUrl.page === "number") {
        const { min = 1, max } = validationConfig.page;

        // Ensure page number is not less than minimum
        if (stateFromUrl.page < min) {
          stateFromUrl.page = min;
        }

        // If maximum is provided, ensure page number does not exceed maximum
        if (max !== undefined && stateFromUrl.page > max) {
          stateFromUrl.page = max;
        }

        // If total pages is provided, ensure page number does not exceed total pages
        if (
          validationConfig.totalPages !== undefined &&
          stateFromUrl.page > validationConfig.totalPages
        ) {
          stateFromUrl.page = Math.max(1, validationConfig.totalPages);
        }
      }

      // Validate per page count
      if (
        validationConfig.pageSize &&
        typeof stateFromUrl.pageSize === "number"
      ) {
        const { min, max } = validationConfig.pageSize;

        // Apply minimum and maximum limits
        if (min !== undefined && stateFromUrl.pageSize < min) {
          stateFromUrl.pageSize = min;
        }
        if (max !== undefined && stateFromUrl.pageSize > max) {
          stateFromUrl.pageSize = max;
        }
      }
    }

    return stateFromUrl;
  };

  const [state, setState] = useState(getInitialStateFromUrl());

  // Validate and adjust state
  const validateState = useCallback(
    (newState: typeof state) => {
      if (!validationConfig) return newState;

      const validatedState = { ...newState };

      // Validate page number
      if (validationConfig.page && typeof validatedState.page === "number") {
        const { min = 1, max } = validationConfig.page;

        // Ensure page number is not less than minimum
        if (validatedState.page < min) {
          validatedState.page = min;
        }

        // If maximum is provided, ensure page number does not exceed maximum
        if (max !== undefined && validatedState.page > max) {
          validatedState.page = max;
        }

        // If total pages is provided, ensure page number does not exceed total pages
        if (
          validationConfig.totalPages !== undefined &&
          validatedState.page > validationConfig.totalPages
        ) {
          validatedState.page = Math.max(1, validationConfig.totalPages);
        }
      }

      // Validate per page count
      if (
        validationConfig.pageSize &&
        typeof validatedState.pageSize === "number"
      ) {
        const { min, max } = validationConfig.pageSize;

        // Apply minimum and maximum limits
        if (min !== undefined && validatedState.pageSize < min) {
          validatedState.pageSize = min;
        }
        if (max !== undefined && validatedState.pageSize > max) {
          validatedState.pageSize = max;
        }
      }

      return validatedState;
    },
    [validationConfig],
  );

  // Update URL when state changes
  useEffect(() => {
    const newParams = new URLSearchParams(searchParams);

    // Handle all state parameters
    Object.entries(state).forEach(([key, value]) => {
      if (value === undefined || value === null || value === "") {
        newParams.delete(key);
      } else if (typeof value === "object") {
        newParams.set(key, JSON.stringify(value));
      } else {
        newParams.set(key, String(value));
      }
    });

    setSearchParams(newParams);
  }, [state, setSearchParams]);

  // Update state when URL parameter changes
  useEffect(() => {
    const nextState = getInitialStateFromUrl();
    setState((current) => {
	  // Manual deep check for simple properties to avoid reference/order issues
	  const isSame = 
		current.page === nextState.page &&
		current.pageSize === nextState.pageSize &&
		current.search === nextState.search &&
		current.orderBy === nextState.orderBy &&
		current.order === nextState.order;
      return isSame ? current : nextState;
	});
  }, [searchParams.toString()]);

  // Check if page number needs to fall back when totalPages changes
  useEffect(() => {
    if (
      validationConfig?.totalPages !== undefined &&
      typeof state.page === "number"
    ) {
      if (state.page > validationConfig.totalPages) {
        setState((prev) => ({
          ...prev,
          page: Math.max(1, validationConfig.totalPages!),
        }));
      }
    }
  }, [validationConfig?.totalPages, state.page]);

  // Provide dedicated methods for updating individual state properties
  const setPage = useCallback(
    (page: number) => {
      setState((prev) => {
        const newState = { ...prev, page };

        return validateState(newState);
      });
    },
    [validateState],
  );

  const setPageSize = useCallback(
    (pageSize: number) => {
      setState((prev) => {
        const newState = { ...prev, pageSize, page: 1 }; // Reset to first page

        return validateState(newState);
      });
    },
    [validateState],
  );

  const setSearch = useCallback(
    (search: string) => {
      setState((prev) => {
        const newState = { ...prev, search, page: 1 }; // Reset to first page

        return validateState(newState);
      });
    },
    [validateState],
  );

  const setOrderBy = useCallback(
    (orderBy: SortType) => {
      setState((prev) => {
        const newState = { ...prev, orderBy };

        return validateState(newState);
      });
    },
    [validateState],
  );

  const setOrder = useCallback(
    (order: string) => {
      setState((prev) => {
        const newState = { ...prev, order };

        return validateState(newState);
      });
    },
    [validateState],
  );

  const setCustomParam = useCallback(
    (key: string, value: any) => {
      setState((prev) => {
        const newState = { ...prev, [key]: value };

        return validateState(newState);
      });
    },
    [validateState],
  );

  return {
    ...state,
    setState: useCallback(
      (newState: typeof state) => {
        setState(validateState(newState));
      },
      [validateState],
    ),
    setPage,
    setPageSize,
    setSearch,
    setOrderBy,
    setOrder,
    setCustomParam,
  };
}

export default useUrlState;
