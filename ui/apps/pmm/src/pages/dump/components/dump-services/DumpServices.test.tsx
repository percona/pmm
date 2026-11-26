import { fireEvent, render, screen } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { DumpServices } from './DumpServices';

const renderServices = (serviceNames: string[]) =>
  render(
    <TestWrapper>
      <DumpServices serviceNames={serviceNames} />
    </TestWrapper>
  );

describe('DumpServices', () => {
  it('shows All services for an empty list', () => {
    renderServices([]);

    expect(screen.getByText('All services')).toBeVisible();
  });

  it('shows two services and a +N chip', async () => {
    renderServices(['a', 'b', 'c', 'd']);

    expect(screen.getByText('a')).toBeVisible();
    expect(screen.getByText('b')).toBeVisible();
    expect(screen.queryByText('c')).not.toBeInTheDocument();

    fireEvent.mouseOver(screen.getByText('+2'));

    expect(await screen.findByText('c')).toBeVisible();
    expect(screen.getByText('d')).toBeVisible();
  });
});
