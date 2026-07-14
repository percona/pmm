import { cleanup, render, screen, waitFor } from '@testing-library/react';
import HighAvailabilityIcon from './HighAvailabilityIcon';
import { HAHealth } from 'types/ha.types';

describe('HighAvailabilityIcon', () => {
  it('should render the main icon', async () => {
    render(<HighAvailabilityIcon health="healthy" />);

    await waitFor(() =>
      expect(screen.queryByTestId('ha-icon')).toBeInTheDocument()
    );
  });

  it('should render the health icon for all types except healthy', async () => {
    const healthTypes: HAHealth[] = ['degraded', 'critical', 'unreachable'];

    for (const health of healthTypes) {
      render(<HighAvailabilityIcon health={health} />);

      await waitFor(() =>
        expect(screen.queryByTestId('ha-health-icon')).toBeInTheDocument()
      );

      cleanup();
    }
  });

  it('should not render the health icon for healthy', async () => {
    render(<HighAvailabilityIcon health="healthy" />);

    await waitFor(() =>
      expect(screen.queryByTestId('ha-health-icon')).not.toBeInTheDocument()
    );
  });
});
