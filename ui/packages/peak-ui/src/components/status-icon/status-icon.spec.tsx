import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { baseThemeOptions, ThemeContextProvider } from '../../design';
import StatusIcon from './status-icon';
import { type StatusIconSeverity, type StatusIconSize } from './status-icon.types';

const renderWithTheme = (ui: React.ReactElement) =>
  render(<ThemeContextProvider themeOptions={baseThemeOptions}>{ui}</ThemeContextProvider>);

const SEVERITIES: StatusIconSeverity[] = ['success', 'info', 'indeterminate', 'warning', 'error'];
const SIZES: StatusIconSize[] = ['medium', 'small'];

describe('StatusIcon', () => {
  it.each(SEVERITIES)('renders severity "%s" for every size', (severity) => {
    SIZES.forEach((size) => {
      const testId = `status-icon-${severity}-${size}`;
      renderWithTheme(<StatusIcon severity={severity} size={size} data-testid={testId} />);
      expect(screen.getByTestId(testId)).toBeInTheDocument();
    });
  });

  it('is decorative (aria-hidden, non-focusable) without titleAccess', () => {
    const { container } = renderWithTheme(<StatusIcon severity="success" />);
    const svg = container.querySelector('svg');
    expect(svg).toHaveAttribute('aria-hidden', 'true');
    expect(svg).toHaveAttribute('focusable', 'false');
    expect(svg).not.toHaveAttribute('role');
  });

  it('exposes an accessible name when titleAccess is provided', () => {
    renderWithTheme(<StatusIcon severity="error" titleAccess="Something failed" />);
    const svg = screen.getByRole('img', { name: 'Something failed' });
    expect(svg).not.toHaveAttribute('aria-hidden');
    expect(svg).toHaveAttribute('focusable', 'false');
  });
});
