import { render, screen } from '@testing-library/react';
import AlertSeverity from './AlertSeverity';

describe('AlertSeverity', () => {
  it.each([
    ['emergency', 'critical', 'Critical'],
    ['error', 'error', 'Error'],
    ['warning', 'warning', 'Warning'],
    ['debug', 'info', 'Info'],
  ])('renders %s with the %s band signifier', (severity, band, label) => {
    render(<AlertSeverity severity={severity} />);

    expect(screen.getByTestId(`alert-severity-${band}`)).toHaveTextContent(
      label
    );
    expect(screen.queryByText(/emergency|debug/i)).toBeNull();
  });

  it('renders an unrecognized severity as plain text', () => {
    render(<AlertSeverity severity="custom" />);

    expect(screen.getByText('Custom')).toBeInTheDocument();
    expect(screen.queryByTestId(/^alert-severity-/)).toBeNull();
  });
});
