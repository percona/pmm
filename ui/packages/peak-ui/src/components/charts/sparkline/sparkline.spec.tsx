import { createRef } from 'react';
import { act, fireEvent, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderInTheme, testTheme } from '../../../test-utils';
import Sparkline from './sparkline';

const data = [3, 7, 2, 9].map((value, index) => ({
  timestamp: index * 60_000,
  value,
}));
const percent = (value: number) => `${value}%`;

describe('Sparkline', () => {
  it('summarizes the latest, lowest and highest value for screen readers', () => {
    renderInTheme(
      <Sparkline data={data} width={200} valueFormatter={percent} />
    );

    expect(screen.getByRole('group')).toHaveAccessibleName(
      'Latest 9%, lowest 2%, highest 9%'
    );
  });

  it('lets keyboard users focus the chart to step through its points', () => {
    const { container } = renderInTheme(<Sparkline data={data} width={200} />);

    expect(
      container.querySelector('[role="group"] [tabindex="0"]')
    ).toBeInTheDocument();
  });

  it('shows the time and value in a tooltip without an empty header cell', async () => {
    renderInTheme(
      <Sparkline
        data={data}
        width={200}
        valueFormatter={percent}
        timeFormatter={(time) => `minute ${time.getTime() / 60_000}`}
      />
    );
    const proxy = document.querySelector<HTMLElement>(
      '[role="group"] [tabindex="0"]'
    )!;

    await act(async () => {
      proxy.focus();
      fireEvent.keyDown(proxy, { key: 'ArrowRight' });
    });

    const tooltip = await screen.findByText('minute 0');
    const paper = tooltip.closest('.MuiChartsTooltip-paper')!;
    expect(paper).toHaveTextContent('3%');
    expect(paper.querySelector('th')).not.toBeInTheDocument();
  });

  it('announces the given label instead of the summary', () => {
    renderInTheme(
      <Sparkline data={data} width={200} ariaLabel="Load over the last hour" />
    );

    expect(screen.getByRole('group')).toHaveAccessibleName(
      'Load over the last hour'
    );
  });

  it('hides a decorative chart from assistive technology', () => {
    const { container } = renderInTheme(
      <Sparkline data={data} width={200} decorative data-testid="spark" />
    );

    expect(screen.queryByRole('group')).not.toBeInTheDocument();
    expect(screen.getByTestId('spark')).toHaveAttribute('aria-hidden', 'true');
    expect(container.querySelector('[tabindex="0"]')).not.toBeInTheDocument();
  });

  it.each(['bar', 'area'] as const)(
    'draws the %s variant with x-charts',
    (variant) => {
      const { container } = renderInTheme(
        <Sparkline data={data} width={200} variant={variant} />
      );

      expect(container.querySelector('svg')).toBeInTheDocument();
      expect(container.querySelector('[data-flat]')).not.toBeInTheDocument();
    }
  );

  it('draws a dotted baseline in the mark color when every value is 0', () => {
    const flat = data.map((point) => ({ ...point, value: 0 }));
    const { container } = renderInTheme(<Sparkline data={flat} height={40} />);

    expect(screen.getByRole('img')).toHaveAccessibleName(
      'Latest 0, lowest 0, highest 0'
    );
    const baseline = container.querySelector('[data-flat]');
    expect(baseline).toHaveStyle({ height: '40px' });
    expect(baseline?.firstElementChild).toHaveStyle({
      borderBottom: `2px dotted ${testTheme.palette.primary.main}`,
    });
    expect(container.querySelector('svg')).not.toBeInTheDocument();
  });

  it('says there is no data when there are no points, without a zero baseline', () => {
    const { container } = renderInTheme(<Sparkline data={[]} />);

    expect(screen.getByRole('img')).toHaveAccessibleName('No data');
    expect(container.querySelector('[data-flat]')).not.toBeInTheDocument();
  });

  it('draws a single point of the area variant as a bar', () => {
    const { container } = renderInTheme(
      <Sparkline data={data.slice(0, 1)} width={200} variant="area" />
    );

    expect(container.querySelector('.MuiBarChart-element')).toBeInTheDocument();
  });

  it('forwards the ref and passes other props to the root', () => {
    const ref = createRef<HTMLDivElement>();
    renderInTheme(<Sparkline ref={ref} data={data} width={200} id="trend" />);

    expect(ref.current).toBe(screen.getByRole('group'));
    expect(ref.current).toHaveAttribute('id', 'trend');
  });
});
