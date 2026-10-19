import { fireEvent, render, screen } from '@testing-library/react';
import type { HubHealth } from '../hub.types';
import { minutesAgo } from '../hub.testFixtures';
import { TestWrapper } from 'utils/testWrapper';
import { HealthChip } from './HealthChip';

const renderChip = (health: HubHealth, since?: string, detail?: string) =>
  render(<HealthChip health={health} since={since} detail={detail} />, {
    wrapper: TestWrapper,
  });

describe('HealthChip', () => {
  it.each<[HubHealth, string, string, string]>([
    ['healthy', 'Healthy', 'MuiChip-colorSuccess', 'MuiChip-outlined'],
    ['degraded', 'Degraded', 'MuiChip-colorWarning', 'MuiChip-filled'],
    ['down', 'Down', 'MuiChip-colorError', 'MuiChip-filled'],
    ['noData', 'No data', 'MuiChip-colorDefault', 'MuiChip-filled'],
  ])('shows %s as the word %s', (health, word, color, variant) => {
    renderChip(health);

    const chip = screen.getByTestId('hub-health');

    expect(chip).toHaveTextContent(word);
    expect(chip).toHaveClass(color, variant);
  });

  it('never shows no data in the healthy color', () => {
    renderChip('noData', minutesAgo(9));

    expect(screen.getByTestId('hub-health')).not.toHaveClass(
      'MuiChip-colorSuccess'
    );
  });

  it('says how long a problem has lasted, with the exact time in a focusable tooltip', async () => {
    renderChip('down', minutesAgo(12));

    const duration = screen.getByText('for 12m');

    expect(duration).toHaveAttribute('tabindex', '0');

    fireEvent.mouseOver(duration);

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      /^Down since /
    );
  });

  it('uses the detail as the tooltip when there is one', async () => {
    renderChip('down', minutesAgo(3), 'No primary since 14:02');

    fireEvent.mouseOver(screen.getByText('for 3m'));

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'No primary since 14:02'
    );
  });

  it('shows no duration while healthy', () => {
    renderChip('healthy', minutesAgo(5));

    expect(screen.queryByText(/^for /)).not.toBeInTheDocument();
  });
});
