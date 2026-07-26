import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import * as advisorsApi from 'api/advisors';
import * as servicesApi from 'api/services';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import {
  AdvisorCheckRow,
  AdvisorTechnology,
  AdvisorInterval,
} from 'types/advisors.types';
import { MySqlService } from 'types/services.types';
import { DisableServicesDrawer } from './DisableServicesDrawer';
import { Messages } from './DisableServicesDrawer.messages';

vi.mock('api/advisors');
vi.mock('api/services');

const mysqlService = (id: string, name: string): MySqlService => ({
  serviceId: id,
  serviceName: name,
  nodeId: 'node-1',
  environment: '',
  cluster: '',
  replicationSet: '',
  customLabels: {},
  address: '127.0.0.1',
  port: 3306,
  socket: '',
  version: '8.0',
  extraDsnParams: {},
});

const TEST_CHECK: AdvisorCheckRow = {
  checkName: 'mysql_version_check',
  summary: 'MySQL version check',
  description: 'Warns if MySQL version is EOL',
  category: 'Configuration',
  subcategory: 'Version',
  technology: AdvisorTechnology.mysql,
  interval: AdvisorInterval.standard,
  enabled: true,
  userDefined: false,
  disabledServiceIds: [],
};

const renderDrawer = (check: AdvisorCheckRow | null, onClose = vi.fn()) =>
  render(
    wrapWithQueryProvider(
      wrapWithSnackbarProvider(
        <DisableServicesDrawer check={check} onClose={onClose} />
      )
    )
  );

const openPicker = async () => {
  const picker = await screen.findByTestId('disable-services-picker');
  const input = within(picker).getByRole('combobox');
  // ArrowDown opens the MUI Autocomplete popup reliably in jsdom
  fireEvent.keyDown(input, { key: 'ArrowDown' });
  return screen.findByRole('listbox', { hidden: true });
};

describe('DisableServicesDrawer', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(servicesApi.listServices).mockResolvedValue({
      mysql: [
        mysqlService('svc-1', 'mysql-svc-1'),
        mysqlService('svc-2', 'mysql-svc-2'),
      ],
    });
    vi.mocked(advisorsApi.changeAdvisorChecks).mockResolvedValue();
  });

  it('disables the check for the selected services', async () => {
    renderDrawer(TEST_CHECK);

    const listbox = await openPicker();
    fireEvent.click(within(listbox).getByText('mysql-svc-1'));

    fireEvent.click(screen.getByTestId('disable-services-submit'));

    await waitFor(() =>
      expect(advisorsApi.changeAdvisorChecks).toHaveBeenCalledWith(
        [
          {
            name: 'mysql_version_check',
            serviceIds: ['svc-1'],
            enable: false,
          },
        ],
        expect.anything()
      )
    );
  });

  it('requests services of the check technology type only', async () => {
    renderDrawer(TEST_CHECK);

    await waitFor(() =>
      expect(servicesApi.listServices).toHaveBeenCalledWith({
        serviceType: 'SERVICE_TYPE_MYSQL_SERVICE',
      })
    );
  });

  it('offers only services the check is not already disabled for', async () => {
    renderDrawer({ ...TEST_CHECK, disabledServiceIds: ['svc-2'] });

    const listbox = await openPicker();
    expect(within(listbox).getByText('mysql-svc-1')).toBeInTheDocument();
    expect(within(listbox).queryByText('mysql-svc-2')).not.toBeInTheDocument();
  });

  it('re-enables the check for a disabled service', async () => {
    renderDrawer({ ...TEST_CHECK, disabledServiceIds: ['svc-2'] });

    // the disabled service is listed by name
    const list = await screen.findByTestId('disable-services-list');
    await waitFor(() =>
      expect(within(list).getByText('mysql-svc-2')).toBeInTheDocument()
    );

    fireEvent.click(screen.getByTestId('disable-services-enable-svc-2'));

    await waitFor(() =>
      expect(advisorsApi.changeAdvisorChecks).toHaveBeenCalledWith(
        [
          {
            name: 'mysql_version_check',
            serviceIds: ['svc-2'],
            enable: true,
          },
        ],
        expect.anything()
      )
    );
  });

  it('blocks adding services when the check is disabled globally but keeps the list', async () => {
    renderDrawer({
      ...TEST_CHECK,
      enabled: false,
      disabledServiceIds: ['svc-2'],
    });

    expect(
      await screen.findByTestId('disable-services-globally-disabled')
    ).toHaveTextContent(Messages.disabledGlobally);
    expect(screen.getByTestId('disable-services-submit')).toBeDisabled();
    const picker = screen.getByTestId('disable-services-picker');
    expect(within(picker).getByRole('combobox')).toBeDisabled();

    // existing per-service settings remain visible and removable
    const list = screen.getByTestId('disable-services-list');
    await waitFor(() =>
      expect(within(list).getByText('mysql-svc-2')).toBeInTheDocument()
    );
    expect(screen.getByTestId('disable-services-enable-svc-2')).toBeEnabled();
  });

  it('falls back to the service ID for services that no longer exist', async () => {
    renderDrawer({ ...TEST_CHECK, disabledServiceIds: ['gone-svc'] });

    const list = await screen.findByTestId('disable-services-list');
    expect(
      within(list).getByText(Messages.removedService('gone-svc'))
    ).toBeInTheDocument();
  });
});
