import { fireEvent, screen } from '@testing-library/react';
import { renderHub } from '../../testUtils';
import { testDoor } from '../hub.testFixtures';
import type { HubMeasure } from '../hub.types';
import { MeasureValue } from './MeasureValue';

const renderMeasure = (measure: HubMeasure) =>
  renderHub(<MeasureValue measure={measure} />);

describe('MeasureValue', () => {
  it('shows a fine value as plain text without a status icon', () => {
    const { container } = renderMeasure({
      kind: 'value',
      seconds: 2,
      level: 'ok',
      door: testDoor('2s', '/graph/d/replset'),
    });

    expect(screen.getByRole('link', { name: '2s' })).toBeInTheDocument();
    expect(container.querySelector('svg')).toBeNull();
  });

  it('marks a value past its threshold with an icon and words', async () => {
    const { container } = renderMeasure({
      kind: 'value',
      seconds: 40,
      level: 'warning',
      rule: 'Warning above 30s sustained for 2m',
      door: testDoor('40s', '/graph/d/replset', {
        context: 'for orders-shard1-m3',
      }),
    });

    const link = screen.getByRole('link', {
      name: '40s over the warning threshold for orders-shard1-m3',
    });

    expect(container.querySelector('svg')).toHaveAttribute(
      'aria-hidden',
      'true'
    );

    fireEvent.mouseOver(link);

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'Warning above 30s sustained for 2m'
    );
  });

  it('renders a dash with its reason when the measure does not apply', () => {
    renderMeasure({
      kind: 'notApplicable',
      reason: 'Standalones do not replicate',
    });

    expect(screen.getByText('—')).toHaveAttribute('aria-hidden', 'true');
    expect(
      screen.getByText('Standalones do not replicate')
    ).toBeInTheDocument();
  });

  it('says No data instead of a value it does not have', () => {
    renderMeasure({ kind: 'noData' });

    expect(screen.getByText('No data')).toBeInTheDocument();
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });
});
