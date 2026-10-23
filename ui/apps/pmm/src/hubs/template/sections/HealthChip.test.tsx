import { fireEvent, render, screen, within } from '@testing-library/react';
import type { HubCondition, HubHealth } from '../hub.types';
import { minutesAgo } from '../hub.testFixtures';
import { TestWrapper } from 'utils/testWrapper';
import { HealthChip } from './HealthChip';

const renderChip = (
  health: HubHealth,
  since?: string,
  conditions?: HubCondition[]
) =>
  render(<HealthChip health={health} since={since} conditions={conditions} />, {
    wrapper: TestWrapper,
  });

describe('HealthChip', () => {
  it.each<[HubHealth, string, string, string]>([
    ['healthy', 'Healthy', 'MuiChip-colorSuccess', 'MuiChip-outlined'],
    ['degraded', 'Degraded', 'MuiChip-colorWarning', 'MuiChip-filled'],
    ['down', 'Down', 'MuiChip-colorError', 'MuiChip-filled'],
    ['upgrading', 'Upgrading', 'MuiChip-colorInfo', 'MuiChip-outlined'],
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

  it('says why beneath the chip, and how many more reasons hold', () => {
    renderChip('degraded', undefined, [
      {
        text: 'orders-shard2-m2 down',
        rule: 'Degraded when a member is down while the set keeps its primary and majority',
      },
      { text: 'orders-shard1-m3 40s behind', rule: 'Degraded when …' },
    ]);

    const reason = screen.getByTestId('hub-health-reason');

    expect(reason).toHaveTextContent('orders-shard2-m2 down · 1 more');
    expect(reason).toHaveAttribute('tabindex', '0');
  });

  it('adds each rule and the exact time on hover', async () => {
    renderChip('down', minutesAgo(3), [
      {
        text: 'No primary · 2 of 3 down',
        rule: "Down when members answer but none is primary, so the set can't take writes",
      },
    ]);

    fireEvent.mouseOver(screen.getByTestId('hub-health-reason'));

    const tooltip = await screen.findByRole('tooltip');

    expect(
      within(tooltip).getByText('No primary · 2 of 3 down')
    ).toBeInTheDocument();
    expect(
      within(tooltip).getByText(
        "Down when members answer but none is primary, so the set can't take writes"
      )
    ).toBeInTheDocument();
    expect(tooltip).toHaveTextContent(/Down since /);
  });

  it('shows no reason without conditions', () => {
    renderChip('noData', minutesAgo(9));

    expect(screen.queryByTestId('hub-health-reason')).toBeNull();
  });

  it('shows no duration while healthy', () => {
    renderChip('healthy', minutesAgo(5));

    expect(screen.queryByText(/^for /)).not.toBeInTheDocument();
  });
});
