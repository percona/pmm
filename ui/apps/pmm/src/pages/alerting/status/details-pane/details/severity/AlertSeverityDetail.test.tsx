import { render, screen } from '@testing-library/react';
import AlertSeverityDetail from './AlertSeverityDetail';

describe('AlertSeverityDetail', () => {
  it('renders unavailable text when severity is missing', () => {
    render(<AlertSeverityDetail />);

    expect(screen.getByText('Unavailable')).toBeInTheDocument();
  });

  it('renders a provided severity', () => {
    render(<AlertSeverityDetail severity="critical" />);

    expect(screen.getByText('Critical')).toBeInTheDocument();
    expect(screen.queryByText(/reported as/i)).toBeNull();
  });

  it('shows the original level when the band hides it', () => {
    render(<AlertSeverityDetail severity="EMERGENCY" />);

    expect(screen.getByText('Critical')).toBeInTheDocument();
    expect(screen.getByText('Reported as Emergency')).toBeInTheDocument();
  });

  it('does not repeat an unrecognized severity', () => {
    render(<AlertSeverityDetail severity="custom" />);

    expect(screen.getByText('Custom')).toBeInTheDocument();
    expect(screen.queryByText(/reported as/i)).toBeNull();
  });
});
