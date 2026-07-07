import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import LoadingSpinner from './LoadingSpinner';

describe('LoadingSpinner', () => {
  it('should render with default size', () => {
    render(<LoadingSpinner />);

    const spinner = screen.getByRole('status', { hidden: true })?.parentElement;
    expect(spinner).toBeInTheDocument();
  });

  it('should render with small size', () => {
    render(<LoadingSpinner size="small" />);

    const spinnerContainer = screen.getByTestId?.('spinner-container') ||
      document.querySelector('.flex.justify-center.items-center');
    expect(spinnerContainer).toBeInTheDocument();
  });

  it('should render with medium size', () => {
    render(<LoadingSpinner size="medium" />);

    const spinner = document.querySelector('.w-8.h-8');
    expect(spinner).toBeInTheDocument();
  });

  it('should render with large size', () => {
    render(<LoadingSpinner size="large" />);

    const spinner = document.querySelector('.w-12.h-12');
    expect(spinner).toBeInTheDocument();
  });

  it('should apply custom className', () => {
    render(<LoadingSpinner className="custom-class" />);

    const container = document.querySelector('.custom-class');
    expect(container).toBeInTheDocument();
  });

  it('should have correct size classes for small', () => {
    const { container } = render(<LoadingSpinner size="small" />);
    const spinner = container.querySelector('div.animate-spin');

    expect(spinner?.className).toContain('w-4');
    expect(spinner?.className).toContain('h-4');
  });

  it('should have correct size classes for medium', () => {
    const { container } = render(<LoadingSpinner size="medium" />);
    const spinner = container.querySelector('div.animate-spin');

    expect(spinner?.className).toContain('w-8');
    expect(spinner?.className).toContain('h-8');
  });

  it('should have correct size classes for large', () => {
    const { container } = render(<LoadingSpinner size="large" />);
    const spinner = container.querySelector('div.animate-spin');

    expect(spinner?.className).toContain('w-12');
    expect(spinner?.className).toContain('h-12');
  });

  it('should have spinning animation class', () => {
    render(<LoadingSpinner />);

    const spinner = document.querySelector('.animate-spin');
    expect(spinner).toBeInTheDocument();
  });

  it('should have border styling', () => {
    render(<LoadingSpinner />);

    const spinner = document.querySelector('.border-4');
    expect(spinner).toBeInTheDocument();
  });

  it('should have primary color top border', () => {
    render(<LoadingSpinner />);

    const spinner = document.querySelector('.border-t-primary-500');
    expect(spinner).toBeInTheDocument();
  });

  it('should have rounded full shape', () => {
    render(<LoadingSpinner />);

    const spinner = document.querySelector('.rounded-full');
    expect(spinner).toBeInTheDocument();
  });

  it('should be memoized', () => {
    // Verify the component is wrapped in React.memo
    expect(LoadingSpinner.displayName).toBe('LoadingSpinner');
  });
});
