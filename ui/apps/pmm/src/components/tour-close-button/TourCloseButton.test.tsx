import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import TourCloseButton from './TourCloseButton';

describe('TourCloseButton', () => {
  const mockEndTour = vi.fn();

  beforeEach(() => {
    mockEndTour.mockClear();
  });

  it('renders the close button', () => {
    render(<TourCloseButton endTour={mockEndTour} />);

    expect(screen.getByTestId('tour-close-button')).toBeDefined();
  });

  it('calls endTour when clicked', () => {
    render(<TourCloseButton endTour={mockEndTour} />);

    fireEvent.click(screen.getByTestId('tour-close-button'));

    expect(mockEndTour).toHaveBeenCalledTimes(1);
  });
});
