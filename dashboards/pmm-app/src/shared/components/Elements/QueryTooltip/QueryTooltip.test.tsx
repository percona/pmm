import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { QueryTooltip } from './QueryTooltip';

jest.mock('highlight.js', () => ({
  highlightElement: jest.fn(),
}));

describe('QueryTooltip::', () => {
  it('renders in the opposite theme to the page', async () => {
    render(
      <QueryTooltip query="SELECT 1" queryId="42">
        <span>Query</span>
      </QueryTooltip>,
    );

    fireEvent.mouseEnter(screen.getByText('Query'));

    expect(await screen.findByText('Query Id: 42')).toHaveStyle({ color: 'rgba(36, 41, 46, 1)' });
    expect(screen.getByTestId('highlight-pre')).toHaveStyle({ backgroundColor: '#ffffff' });
  });
});
