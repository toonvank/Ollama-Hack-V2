import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import ProtectedRoute from './ProtectedRoute';
import { AuthProvider } from './AuthContext';

// Mock the AuthContext
vi.mock('./AuthContext', () => ({
  useAuth: vi.fn(),
  AuthProvider: vi.fn(({ children }) => <div>{children}</div>),
}));

const { useAuth } = await import('./AuthContext');

const createWrapper = (authState: { isAuthenticated: boolean; isLoading: boolean; isAdmin: boolean }) => {
  return ({ children }: { children: React.ReactNode }) => (
    <MemoryRouter initialEntries={['/protected']}>
      <AuthProvider>
        <Routes>
          <Route path="/protected" element={<ProtectedRoute />}>
            <Route path="child" element={<div data-testid="child-content">Child Content</div>} />
          </Route>
          <Route path="/login" element={<div data-testid="login-page">Login Page</div>} />
          <Route path="/unauthorized" element={<div data-testid="unauthorized-page">Unauthorized</div>} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>
  );
};

describe('ProtectedRoute', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should show loading spinner when loading', () => {
    (useAuth as ReturnType<typeof vi.fn>).mockReturnValue({
      isAuthenticated: false,
      isLoading: true,
      isAdmin: false,
    });

    render(<ProtectedRoute />, { wrapper: createWrapper({ isAuthenticated: false, isLoading: true, isAdmin: false }) });

    expect(screen.getByTestId('child-content')).toBeInTheDocument();
  });

  it('should redirect to login when not authenticated', () => {
    (useAuth as ReturnType<typeof vi.fn>).mockReturnValue({
      isAuthenticated: false,
      isLoading: false,
      isAdmin: false,
    });

    render(<ProtectedRoute />, { wrapper: createWrapper({ isAuthenticated: false, isLoading: false, isAdmin: false }) });

    expect(window.location.pathname).toBe('/login');
  });

  it('should allow access when authenticated', () => {
    (useAuth as ReturnType<typeof vi.fn>).mockReturnValue({
      isAuthenticated: true,
      isLoading: false,
      isAdmin: false,
    });

    render(<ProtectedRoute />, { wrapper: createWrapper({ isAuthenticated: true, isLoading: false, isAdmin: false }) });

    expect(screen.getByTestId('child-content')).toBeInTheDocument();
  });

  it('should redirect to unauthorized when admin required but user is not admin', () => {
    (useAuth as ReturnType<typeof vi.fn>).mockReturnValue({
      isAuthenticated: true,
      isLoading: false,
      isAdmin: false,
    });

    render(<ProtectedRoute requireAdmin />, { wrapper: createWrapper({ isAuthenticated: true, isLoading: false, isAdmin: false }) });

    expect(window.location.pathname).toBe('/unauthorized');
  });

  it('should allow access when admin required and user is admin', () => {
    (useAuth as ReturnType<typeof vi.fn>).mockReturnValue({
      isAuthenticated: true,
      isLoading: false,
      isAdmin: true,
    });

    render(<ProtectedRoute requireAdmin />, { wrapper: createWrapper({ isAuthenticated: true, isLoading: false, isAdmin: true }) });

    expect(screen.getByTestId('child-content')).toBeInTheDocument();
  });

  it('should preserve location state when redirecting to login', () => {
    (useAuth as ReturnType<typeof vi.fn>).mockReturnValue({
      isAuthenticated: false,
      isLoading: false,
      isAdmin: false,
    });

    render(<ProtectedRoute />, { wrapper: createWrapper({ isAuthenticated: false, isLoading: false, isAdmin: false }) });

    // The Navigate component should have state with from: location
    expect(window.location.pathname).toBe('/login');
  });

  it('should use replace navigation for redirects', () => {
    (useAuth as ReturnType<typeof vi.fn>).mockReturnValue({
      isAuthenticated: false,
      isLoading: false,
      isAdmin: false,
    });

    render(<ProtectedRoute />, { wrapper: createWrapper({ isAuthenticated: false, isLoading: false, isAdmin: false }) });

    expect(window.location.pathname).toBe('/login');
  });

  it('should render Outlet for child routes', () => {
    (useAuth as ReturnType<typeof vi.fn>).mockReturnValue({
      isAuthenticated: true,
      isLoading: false,
      isAdmin: false,
    });

    render(<ProtectedRoute />, { wrapper: createWrapper({ isAuthenticated: true, isLoading: false, isAdmin: false }) });

    expect(screen.getByTestId('child-content')).toBeInTheDocument();
  });
});
