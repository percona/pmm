import { act, render, screen } from '@testing-library/react';
import DelayedRender from './DelayedRender';

describe('DelayedRender', () => {
  it('should render children after delay', () => {
    vi.useFakeTimers();

    render(<DelayedRender delay={1000}>Hello</DelayedRender>);

    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(screen.getByText('Hello')).toBeInTheDocument();
  });

  it('should not render children before delay', () => {
    vi.useFakeTimers();

    render(<DelayedRender delay={1000}>Hello</DelayedRender>);

    expect(screen.queryByText('Hello')).not.toBeInTheDocument();
  });
});
