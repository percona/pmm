import { createRef } from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import { describe, expect, it, vi } from 'vitest';
import { getThemeOptions } from '../../design';
import DetailsPane from './details-pane';
import type { DetailsPaneProps } from './details-pane.types';

const tabs: DetailsPaneProps['tabs'] = [
  { value: 'details', label: 'Details', content: <p>Details content</p> },
  {
    value: 'raw-data',
    label: 'Raw data',
    content: <p>Raw content</p>,
    tabProps: { 'data-testid': 'raw-tab' },
  },
];

const navigation = () => ({ isFirst: false, isLast: false, next: vi.fn(), previous: vi.fn() });

const renderPane = (props: Partial<DetailsPaneProps> = {}) => {
  const allProps: DetailsPaneProps = { open: true, onClose: vi.fn(), tabs, ...props };
  const theme = createTheme(getThemeOptions('base')('light'));
  const result = render(
    <ThemeProvider theme={theme}>
      <DetailsPane data-testid="pane" {...allProps} />
    </ThemeProvider>
  );
  const rerender = (next: Partial<DetailsPaneProps>) =>
    result.rerender(
      <ThemeProvider theme={theme}>
        <DetailsPane data-testid="pane" {...allProps} {...next} />
      </ThemeProvider>
    );
  return { ...result, props: allProps, rerender };
};

describe('DetailsPane', () => {
  it('shows the first tab and only its content', () => {
    renderPane();

    expect(screen.getByRole('tab', { name: 'Details' })).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByText('Details content')).toBeInTheDocument();
    expect(screen.queryByText('Raw content')).not.toBeInTheDocument();
  });

  it('switches tabs and reports the change', () => {
    const onTabChange = vi.fn();
    renderPane({ onTabChange });

    fireEvent.click(screen.getByTestId('raw-tab'));

    expect(screen.getByText('Raw content')).toBeInTheDocument();
    expect(onTabChange).toHaveBeenCalledWith('raw-data');
  });

  it('opens on defaultTab and follows a controlled tab', () => {
    const { rerender } = renderPane({ defaultTab: 'raw-data' });
    expect(screen.getByText('Raw content')).toBeInTheDocument();

    rerender({ tab: 'details' });
    expect(screen.getByText('Details content')).toBeInTheDocument();
  });

  it('falls back to the first tab when the requested one is missing', () => {
    renderPane({ tab: 'nope' });

    expect(screen.getByText('Details content')).toBeInTheDocument();
  });

  it('goes back to the default tab after it closes', () => {
    const { rerender } = renderPane();
    fireEvent.click(screen.getByTestId('raw-tab'));
    expect(screen.getByText('Raw content')).toBeInTheDocument();

    rerender({ open: false });
    rerender({ open: true });

    expect(screen.getByText('Details content')).toBeInTheDocument();
  });

  it('hides the pane from assistive technology while closed', () => {
    const { rerender } = renderPane();
    expect(screen.getByTestId('pane')).toHaveAttribute('aria-hidden', 'false');

    rerender({ open: false });
    expect(screen.getByTestId('pane')).toHaveAttribute('aria-hidden', 'true');
  });

  it('closes from the close button and from Escape while open', () => {
    const { props } = renderPane();

    fireEvent.click(screen.getByRole('button', { name: 'Close details' }));
    fireEvent.keyDown(document, { key: 'Escape' });

    expect(props.onClose).toHaveBeenCalledTimes(2);
  });

  it('ignores Escape while closed or when something else handled it', () => {
    const { props, rerender } = renderPane({ open: false });
    fireEvent.keyDown(document, { key: 'Escape' });

    rerender({ open: true });
    const event = new KeyboardEvent('keydown', { key: 'Escape', cancelable: true });
    event.preventDefault();
    document.dispatchEvent(event);

    expect(props.onClose).not.toHaveBeenCalled();
  });

  it('hides the arrows without navigation', () => {
    renderPane();

    expect(screen.queryByRole('button', { name: 'Previous' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Next' })).not.toBeInTheDocument();
  });

  it('steps through rows with the arrows', () => {
    const nav = navigation();
    renderPane({ navigation: nav });

    fireEvent.click(screen.getByRole('button', { name: 'Previous' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));

    expect(nav.previous).toHaveBeenCalledTimes(1);
    expect(nav.next).toHaveBeenCalledTimes(1);
  });

  it('disables the arrows at either end', () => {
    renderPane({ navigation: { ...navigation(), isFirst: true, isLast: true } });

    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled();
  });

  it('names the buttons from labels and passes button props through', () => {
    renderPane({
      navigation: navigation(),
      labels: { previous: 'Previous query', next: 'Next query' },
      previousButtonProps: { 'data-testid': 'prev' },
      nextButtonProps: { 'data-testid': 'next' },
      closeButtonProps: { 'data-testid': 'close', 'aria-label': 'Close details pane' },
    });

    expect(screen.getByTestId('prev')).toHaveAccessibleName('Previous query');
    expect(screen.getByTestId('next')).toHaveAccessibleName('Next query');
    expect(screen.getByTestId('close')).toHaveAccessibleName('Close details pane');
  });

  it('forwards the ref to the pane surface', () => {
    const ref = createRef<HTMLDivElement>();
    render(
      <ThemeProvider theme={createTheme(getThemeOptions('base')('light'))}>
        <DetailsPane ref={ref} data-testid="pane" open onClose={vi.fn()} tabs={tabs} />
      </ThemeProvider>
    );

    expect(ref.current).toBe(screen.getByTestId('pane'));
  });
});
