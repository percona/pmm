import { fireEvent, render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import AutoRefreshSelect from './AutoRefreshSelect';

const onRefreshIntervalChangeMock = vi.fn();

const renderComponent = (isFetching = true) =>
  render(
    <AutoRefreshSelect
      isFetching={isFetching}
      refreshInterval={1000}
      onRefreshIntervalChange={onRefreshIntervalChangeMock}
    />
  );

describe('AutoRefreshSelect', () => {
  it('should render auto refresh select', () => {
    renderComponent();

    expect(screen.getByTestId('auto-refresh-button')).toBeInTheDocument();
  });

  it('should call onRefreshIntervalChange when refresh interval is changed', () => {
    renderComponent();

    fireEvent.click(screen.getByTestId('auto-refresh-button'));

    fireEvent.click(screen.getByTestId('text-select-option-2000'));

    expect(onRefreshIntervalChangeMock).toHaveBeenCalledWith(2000);
  });

  it('should be disabled when isFetching is false', () => {
    renderComponent(false);

    expect(screen.getByTestId('auto-refresh-button')).toBeDisabled();
  });
});
