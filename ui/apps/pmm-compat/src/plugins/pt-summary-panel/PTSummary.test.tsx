import React from 'react';
import { render, screen } from '@testing-library/react';
import type { PanelProps } from '@grafana/data';

import { PTSummaryPanel } from './PTSummary';

describe('PTSummaryPanel::', () => {
  it('Renders correctly with props', () => {
    const props = {
      data: {
        series: [{ fields: [{ name: 'summary', values: ['Test data'] }] }],
      },
    };
    render(<PTSummaryPanel {...(props as unknown as PanelProps)} />);

    expect(screen.getByTestId('pt-summary-fingerprint')?.textContent).toBe(
      'Test data'
    );
  });

  it('Renders correctly without data', () => {
    const props = {
      data: {
        series: [],
      },
    };
    render(<PTSummaryPanel {...(props as unknown as PanelProps)} />);

    expect(screen.getByTestId('pt-summary-fingerprint')?.textContent).toBe('');
  });
});
