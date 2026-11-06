import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { ExportDataset } from './ExportDataset';
import { Messages } from './ExportDataset.messages';

vi.mock('hooks/api/useServices', () => ({
  useServices: () => ({
    data: { mysql: [{ serviceName: 'mysql-1' }] },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

vi.mock('hooks/api/useDump', () => ({
  useStartDump: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

const renderExportDataset = () =>
  render(
    <TestWrapper>
      <ExportDataset />
    </TestWrapper>
  );

describe('ExportDataset', () => {
  it('shows the all services placeholder until a service is selected', async () => {
    renderExportDataset();
    const input = screen.getByTestId('service-select-input');

    expect(input).toHaveAttribute('placeholder', Messages.allServices);

    fireEvent.mouseDown(input);
    fireEvent.click(screen.getByRole('option', { name: 'mysql-1' }));

    await waitFor(() => expect(input).not.toHaveAttribute('placeholder'));
  });
});
