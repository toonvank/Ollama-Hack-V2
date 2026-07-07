import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import ErrorDisplay from './ErrorDisplay';

describe('ErrorDisplay', () => {
  it('should render null when no error', () => {
    const { container } = render(<ErrorDisplay error={null} />);

    expect(container.firstChild).toBeNull();
  });

  it('should render error message', () => {
    const error = new Error('Test error message');

    render(<ErrorDisplay error={error} />);

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    expect(screen.getByText('Test error message')).toBeInTheDocument();
  });

  it('should render with custom className', () => {
    const error = new Error('Test error');

    render(<ErrorDisplay error={error} className="custom-class" />);

    const container = document.querySelector('.custom-class');
    expect(container).toBeInTheDocument();
  });

  it('should render retry button when onRetry is provided', () => {
    const error = new Error('Test error');
    const onRetry = vi.fn();

    render(<ErrorDisplay error={error} onRetry={onRetry} />);

    const retryButton = screen.getByRole('button', { name: /retry/i });
    expect(retryButton).toBeInTheDocument();
  });

  it('should call onRetry when retry button is clicked', () => {
    const error = new Error('Test error');
    const onRetry = vi.fn();

    render(<ErrorDisplay error={error} onRetry={onRetry} />);

    const retryButton = screen.getByRole('button', { name: /retry/i });
    fireEvent.click(retryButton);

    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('should not render retry button when onRetry is not provided', () => {
    const error = new Error('Test error');

    render(<ErrorDisplay error={error} />);

    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('should handle error with empty message', () => {
    const error = new Error('');

    render(<ErrorDisplay error={error} />);

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    expect(screen.getByText('An unexpected error occurred')).toBeInTheDocument();
  });

  it('should have correct danger styling', () => {
    const error = new Error('Test error');

    render(<ErrorDisplay error={error} />);

    const container = document.querySelector('.border-danger-200');
    expect(container).toBeInTheDocument();
  });

  it('should have dark mode support', () => {
    const error = new Error('Test error');

    render(<ErrorDisplay error={error} />);

    const container = document.querySelector('.dark\\:bg-danger-900\\/20');
    expect(container).toBeInTheDocument();
  });

  it('should have warning icon', () => {
    const error = new Error('Test error');

    render(<ErrorDisplay error={error} />);

    expect(screen.getByText('⚠')).toBeInTheDocument();
  });

  it('should have proper layout structure', () => {
    const error = new Error('Test error');

    render(<ErrorDisplay error={error} />);

    const flexContainer = document.querySelector('.flex.items-start.gap-3');
    expect(flexContainer).toBeInTheDocument();
  });

  it('should have proper text styling', () => {
    const error = new Error('Test error');

    render(<ErrorDisplay error={error} />);

    const title = screen.getByText('Something went wrong');
    expect(title.className).toContain('font-medium');
    expect(title.className).toContain('text-danger-700');
  });

  it('should have proper description text styling', () => {
    const error = new Error('Test error');

    render(<ErrorDisplay error={error} />);

    const description = screen.getByText('Test error message');
    expect(description.className).toContain('text-sm');
    expect(description.className).toContain('text-danger-600');
  });

  it('should have proper padding', () => {
    const error = new Error('Test error');

    render(<ErrorDisplay error={error} />);

    const container = document.querySelector('.p-4');
    expect(container).toBeInTheDocument();
  });

  it('should have rounded corners', () => {
    const error = new Error('Test error');

    render(<ErrorDisplay error={error} />);

    const container = document.querySelector('.rounded-lg');
    expect(container).toBeInTheDocument();
  });

  it('should have my-4 margin', () => {
    const error = new Error('Test error');

    render(<ErrorDisplay error={error} />);

    const container = document.querySelector('.my-4');
    expect(container).toBeInTheDocument();
  });
});
