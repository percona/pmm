import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { listDumps, startDump } from 'api/dump';
import { listServices } from 'api/services';
import { DumpStatus } from 'types/dump.types';
import { TEST_DUMP, TEST_SERVICES_WITH_ONE_MYSQL } from 'utils/testStubs';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';
import { ExportDataset } from './ExportDataset';

vi.mock('api/dump');
vi.mock('api/services');

const renderForm = () =>
  render(
    <TestWrapper>
      {wrapWithQueryProvider(wrapWithSnackbarProvider(<ExportDataset />))}
    </TestWrapper>
  );

const createButton = () => screen.getByTestId('create-dataset-submit-button');

describe('ExportDataset', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(listServices).mockResolvedValue(TEST_SERVICES_WITH_ONE_MYSQL);
    vi.mocked(listDumps).mockResolvedValue({ dumps: [TEST_DUMP] });
    vi.mocked(startDump).mockResolvedValue({ dumpId: 'new' });
  });

  it('defaults to all services and does not ignore load', async () => {
    renderForm();

    expect(
      await screen.findByRole('radio', { name: 'All services (1)' })
    ).toBeChecked();
    expect(
      screen.getByLabelText(/^Export even when PMM Server is busy/)
    ).not.toBeChecked();
  });

  it('shows the service picker only for selected services', async () => {
    renderForm();
    expect(
      screen.queryByTestId('service-select-input')
    ).not.toBeInTheDocument();

    fireEvent.click(
      await screen.findByRole('radio', { name: 'Only selected services' })
    );

    expect(screen.getByTestId('service-select-input')).toBeVisible();
  });

  it('sends an empty service list for all services', async () => {
    renderForm();
    await screen.findByRole('radio', { name: 'All services (1)' });
    await waitFor(() => expect(createButton()).toBeEnabled());

    fireEvent.click(createButton());

    await waitFor(() => expect(startDump).toHaveBeenCalled());
    const payload = vi.mocked(startDump).mock.calls[0][0];
    expect(payload).toEqual(
      expect.objectContaining({ serviceNames: [], ignoreLoad: false })
    );
    expect(payload).not.toHaveProperty('serviceScope');
  });

  it('still offers all services when services fail to load', async () => {
    vi.mocked(listServices).mockRejectedValue(new Error('down'));
    renderForm();

    expect(await screen.findByRole('button', { name: 'Retry' })).toBeVisible();
    expect(screen.getByRole('radio', { name: 'All services' })).toBeChecked();
    await waitFor(() => expect(createButton()).toBeEnabled());
  });

  it('blocks creation while another dataset is being created', async () => {
    vi.mocked(listDumps).mockResolvedValue({
      dumps: [{ ...TEST_DUMP, status: DumpStatus.InProgress }],
    });
    renderForm();

    expect(
      await screen.findByText(
        'Another dataset is being created. You can create one when it finishes.'
      )
    ).toBeVisible();
    expect(createButton()).toBeDisabled();
  });

  it("warns that the encryption password can't be recovered", async () => {
    renderForm();

    fireEvent.click(screen.getByLabelText(/^Encrypt the archive/));

    expect(await screen.findByTestId('dump-encryption-password')).toBeVisible();
    expect(
      screen.getByText(/PMM doesn't store this password and can't recover it/)
    ).toBeVisible();
  });
});
