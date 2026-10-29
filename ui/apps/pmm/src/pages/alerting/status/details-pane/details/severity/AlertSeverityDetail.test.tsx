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
  });
});
