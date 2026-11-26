import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderInTheme, testTheme } from '../../../test-utils';
import BarList from './bar-list';

const items = [
  { label: '0-1 ms', value: 0.5 },
  { label: '1-2 ms', value: 0.0635 },
  { label: '2-5 ms', value: 0.25 },
  { label: '5-10 ms', value: 0 },
];

describe('BarList', () => {
  it('reads as a list of labels and values', () => {
    const { container } = renderInTheme(<BarList items={items} />);

    const terms = screen.getAllByRole('term').map((term) => term.textContent);
    const values = screen
      .getAllByRole('definition')
      .map((value) => value.textContent);
    expect(container.firstElementChild?.tagName).toBe('DL');
    expect(terms).toEqual(['0-1 ms', '1-2 ms', '2-5 ms', '5-10 ms']);
    expect(values).toEqual(['50%', '6.4%', '25%', '—No data']);
  });

  it('draws bars relative to the largest value and hides them', () => {
    const { container } = renderInTheme(<BarList items={items} />);

    const bars = Array.from(
      container.querySelectorAll<HTMLElement>('[data-bar]')
    );
    expect(
      bars.map((bar) => bar.style.width || getComputedStyle(bar).width)
    ).toEqual(['80%', '10.16%', '40%']);
    bars.forEach((bar) => expect(bar).toHaveAttribute('aria-hidden', 'true'));
  });

  it('shows the empty glyph but reads "no data" for an empty row', () => {
    renderInTheme(
      <BarList items={items} emptyLabel="n/a" emptyText="Nothing measured" />
    );

    expect(screen.getByText('n/a')).toHaveAttribute('aria-hidden', 'true');
    expect(screen.getByText('Nothing measured')).toHaveStyle({
      position: 'absolute',
    });
  });

  it('formats values and colors bars as asked', () => {
    const { container } = renderInTheme(
      <BarList
        items={[{ label: 'a', value: 12 }]}
        valueFormatter={(v) => `${v} q`}
        color="chart3"
      />
    );

    expect(screen.getByText('12 q')).toBeInTheDocument();
    expect(container.querySelector('[data-bar]')).toHaveStyle({
      backgroundColor: testTheme.palette.charts.chart3,
    });
  });

  it('renders no rows and no bars for no items', () => {
    const { container } = renderInTheme(<BarList items={[]} />);

    expect(container.querySelector('dl')).toBeEmptyDOMElement();
  });
});
