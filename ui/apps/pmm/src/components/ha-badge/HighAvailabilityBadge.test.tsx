import { cleanup, render, screen } from '@testing-library/react';
import HighAvailabilityBadge from './HighAvailabilityBadge';
import { HAHealth } from 'types/ha.types';
import { HIGH_AVAILABILITY_BADGE_HEALTH } from './HighAvailabilityBadge.constants';

describe('HighAvailabilityBadge', () => {
  it('should render the badge', () => {
    render(<HighAvailabilityBadge health="healthy" />);

    expect(screen.getByText('Healthy')).toBeInTheDocument();
  });

  it('should render the badge for all health statuses', () => {
    const healthTypes: HAHealth[] = ['degraded', 'critical', 'unreachable'];

    for (const health of healthTypes) {
      render(<HighAvailabilityBadge health={health} />);

      expect(
        screen.getByText(HIGH_AVAILABILITY_BADGE_HEALTH[health])
      ).toBeInTheDocument();

      cleanup();
    }
  });
});
