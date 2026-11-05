import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { DataFrame, LoadingState, PanelProps } from '@grafana/data';
import { AuroraTreePanel } from './AuroraTreePanel';
import { DEFAULT_OPTIONS } from './AuroraTreePanel.constants';
import { mergedFrame, timeRange } from './AuroraTreePanel.mocks';
import { AuroraTreeOptions } from './AuroraTreePanel.types';

jest.mock('@grafana/runtime', () => ({
  config: { appSubUrl: '/graph' },
}));

const renderPanel = (series: DataFrame[]) => {
  const props = {
    data: { series, state: LoadingState.Done, timeRange },
    options: DEFAULT_OPTIONS,
    timeRange,
  } as unknown as PanelProps<AuroraTreeOptions>;

  return render(<AuroraTreePanel {...props} />);
};

describe('AuroraTreePanel::', () => {
  it('renders clusters collapsed with their aggregates', () => {
    renderPanel([mergedFrame]);

    expect(screen.getByText('database-3')).toBeInTheDocument();
    expect(screen.getByText('AWS cluster unknown')).toBeInTheDocument();
    expect(screen.getAllByText('1 writer')).toHaveLength(2);
    expect(screen.getByText('changing')).toBeInTheDocument();
    expect(screen.getByText('24.75 qps')).toBeInTheDocument();
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });

  it('links every service to its details dashboard once expanded', () => {
    renderPanel([mergedFrame]);

    fireEvent.click(screen.getByRole('button', { name: 'Expand all rows' }));

    expect(screen.getAllByRole('link')).toHaveLength(4);
    expect(screen.getByRole('link', { name: 'database-3-instance-1-reader' })).toHaveAttribute(
      'href',
      '/graph/d/mysql-amazonaurora/mysql-amazon-aurora-details'
      + '?var-service_name=database-3-instance-1-reader&from=now-12h&to=now',
    );
    expect(screen.getByText('reader')).toBeInTheDocument();
    expect(screen.getByText('unknown')).toBeInTheDocument();
    expect(screen.getByText('Down')).toBeInTheDocument();
  });

  it('renders an empty state without data', () => {
    renderPanel([]);

    expect(screen.getByText('No Amazon Aurora MySQL instances found')).toBeInTheDocument();
  });
});
