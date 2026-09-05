import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import {
  ApiError,
  type ConnectivityResult,
  type ConnectivityStatus,
  REDACTED_SECRET,
  SettingClassGroup,
  useConnectivityCheck,
  useResetSetting,
  usePatchSetting,
  useSettingsList,
} from '@sep/api';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithSnackbarProvider } from 'utils/testUtils';
import { Messages } from '../../Settings.messages';
import { ServiceNowConnection } from './ServiceNowConnection';

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  useSettingsList: vi.fn(),
  usePatchSetting: vi.fn(),
  useResetSetting: vi.fn(),
  useConnectivityCheck: vi.fn(),
}));

const settingsList = vi.mocked(useSettingsList);
const patchSetting = vi.mocked(usePatchSetting);
const resetSetting = vi.mocked(useResetSetting);
const connectivityCheck = vi.mocked(useConnectivityCheck);

const patchMutation = vi.fn();
const resetMutation = vi.fn();
const probeMutation = vi.fn();
const probeReset = vi.fn();
const refetch = vi.fn();

const setting = (key: string, value: unknown, hasOverride = false) =>
  ({
    key,
    value,
    has_override: hasOverride,
    default_value: null,
    description: null,
    is_advanced: false,
    is_applicable: true,
    is_complex: true,
    is_secret: false,
    reload: 'none',
    setting_class: 'SEPSettings',
    type: 'object',
  }) as unknown as SettingClassGroup['settings'][number];

const sepGroups = (
  declared: string[],
  storedSecrets?: Record<string, string>,
  endpoint = ''
): SettingClassGroup[] => [
  {
    setting_class: 'SEPSettings',
    is_app_owned: false,
    settings: [
      setting('DIAGNOSTICS_DELIVERY', {
        secrets: Object.fromEntries(
          declared.map((name) => [name, REDACTED_SECRET])
        ),
      }),
      setting(
        'DIAGNOSTICS_DELIVERY_INPUTS',
        { endpoint, secrets: storedSecrets ?? {} },
        storedSecrets !== undefined
      ),
    ],
  },
];

const mockList = (
  overrides: Partial<ReturnType<typeof useSettingsList>> = {}
) => {
  settingsList.mockReturnValue({
    data: sepGroups(['sn_api_key', 'client_token']),
    isLoading: false,
    isFetching: false,
    error: null,
    refetch,
    ...overrides,
  } as unknown as ReturnType<typeof useSettingsList>);
};

const mockPatch = (error: ApiError | null = null) => {
  patchSetting.mockReturnValue({
    mutateAsync: patchMutation,
    error,
  } as unknown as ReturnType<typeof usePatchSetting>);
};

/** Both declared credentials stored, which is what "connected" means here. */
const mockConfigured = (endpoint = '') =>
  mockList({
    data: sepGroups(
      ['sn_api_key', 'client_token'],
      { sn_api_key: REDACTED_SECRET, client_token: REDACTED_SECRET },
      endpoint
    ),
  } as Partial<ReturnType<typeof useSettingsList>>);

const probed = (
  status: ConnectivityStatus,
  reachable = status === 'reachable'
): ConnectivityResult => ({
  service: 'delivery',
  reachable,
  status,
  detail: 'whatever SEP said',
  version: null,
});

/** Loosely typed on purpose: React Query's result union forbids `data` on a
 *  pending mutation, which is exactly the combination a re-test renders. */
const mockProbe = (overrides: Record<string, unknown> = {}) => {
  connectivityCheck.mockReturnValue({
    mutate: probeMutation,
    reset: probeReset,
    data: undefined,
    error: null,
    isPending: false,
    ...overrides,
  } as unknown as ReturnType<typeof useConnectivityCheck>);
};

const renderTab = () =>
  render(
    <TestWrapper>
      {wrapWithSnackbarProvider(<ServiceNowConnection />)}
    </TestWrapper>
  );

const type = (testId: string, value: string) =>
  fireEvent.change(screen.getByTestId(testId), { target: { value } });

const fillCredentials = () => {
  type('servicenow-secret-sn_api_key', 'key-1');
  type('servicenow-secret-client_token', 'token-1');
};

const submit = async () => {
  const button = screen.getByTestId('servicenow-submit');
  await waitFor(() => expect(button).toBeEnabled());
  fireEvent.click(button);
};

beforeEach(() => {
  vi.clearAllMocks();
  patchMutation.mockResolvedValue([]);
  resetMutation.mockResolvedValue(undefined);
  mockList();
  mockPatch();
  mockProbe();
  resetSetting.mockReturnValue({
    mutateAsync: resetMutation,
    isPending: false,
  } as unknown as ReturnType<typeof useResetSetting>);
});

describe('ServiceNowConnection — states', () => {
  it('offers both steps while nothing is connected', () => {
    renderTab();

    expect(
      screen.getByTestId('servicenow-step-credentials')
    ).toBeInTheDocument();
    expect(screen.getByTestId('servicenow-step-connect')).toBeInTheDocument();
    expect(
      screen.getByTestId('servicenow-request-credentials')
    ).toHaveAttribute('href', Messages.serviceNow.requestCredentialsLink);
    expect(
      screen.queryByTestId('servicenow-connected')
    ).not.toBeInTheDocument();
  });

  it('renders one masked field per declared secret name', () => {
    renderTab();

    expect(screen.getByTestId('servicenow-secret-sn_api_key')).toHaveAttribute(
      'type',
      'password'
    );
    expect(
      screen.getByTestId('servicenow-secret-client_token')
    ).toBeInTheDocument();
  });

  it('reveals a credential on request so a pasted value can be checked', () => {
    renderTab();

    fireEvent.click(screen.getByTestId('servicenow-secret-sn_api_key-reveal'));

    expect(screen.getByTestId('servicenow-secret-sn_api_key')).toHaveAttribute(
      'type',
      'text'
    );
  });

  it('follows the plan when an image renames a declared secret', () => {
    mockList({
      data: sepGroups(['sn_api_key', 'renamed_token'], {
        sn_api_key: REDACTED_SECRET,
        client_token: REDACTED_SECRET,
      }),
    } as Partial<ReturnType<typeof useSettingsList>>);

    renderTab();

    expect(
      screen.getByTestId('servicenow-secret-renamed_token')
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId('servicenow-secret-client_token')
    ).not.toBeInTheDocument();
    expect(screen.getByTestId('servicenow-drifted')).toHaveTextContent(
      Messages.serviceNow.driftedWarning
    );
  });

  it('shows a spinner while the settings load', () => {
    mockList({ isLoading: true, data: undefined });
    renderTab();

    expect(screen.getByTestId('servicenow-loading')).toBeInTheDocument();
    expect(screen.queryByTestId('servicenow-submit')).not.toBeInTheDocument();
  });

  it('explains a load that was refused, and offers a retry', () => {
    mockList({
      data: undefined,
      error: new ApiError({ kind: 'http', status: 403, message: 'HTTP 403' }),
    });
    renderTab();

    expect(screen.getByTestId('servicenow-load-error')).toHaveTextContent(
      Messages.serviceNow.errors.forbidden
    );
    expect(screen.queryByTestId('servicenow-submit')).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId('servicenow-load-retry'));
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it('still offers the endpoint when the plan declares no credentials', () => {
    mockList({ data: sepGroups([]) } as Partial<
      ReturnType<typeof useSettingsList>
    >);

    renderTab();

    expect(screen.getByTestId('servicenow-no-secrets')).toBeInTheDocument();
    expect(screen.getByTestId('servicenow-endpoint')).toBeInTheDocument();
  });

  it('offers nothing when the deployment does not carry the key at all', () => {
    mockList({
      data: [
        { setting_class: 'SEPSettings', is_app_owned: false, settings: [] },
      ] as SettingClassGroup[],
    });

    renderTab();

    expect(screen.getByTestId('servicenow-unavailable')).toHaveTextContent(
      Messages.serviceNow.unavailable
    );
    expect(screen.queryByTestId('servicenow-endpoint')).not.toBeInTheDocument();
  });
});

describe('ServiceNowConnection — connected', () => {
  it('reports the connection instead of asking for it again', () => {
    mockConfigured('https://acme.service-now.com/');
    renderTab();

    expect(screen.getByTestId('servicenow-connected')).toHaveTextContent(
      Messages.serviceNow.connectedTitle
    );
    expect(
      screen.getByTestId('servicenow-connected-endpoint')
    ).toHaveTextContent('https://acme.service-now.com/');
    expect(screen.queryByTestId('servicenow-submit')).not.toBeInTheDocument();
  });

  it('names the bundled receiver when no endpoint was supplied', () => {
    mockConfigured();
    renderTab();

    expect(
      screen.getByTestId('servicenow-connected-endpoint')
    ).toHaveTextContent(Messages.serviceNow.defaultEndpoint);
  });

  it('renews through an empty form, never showing the stored masks back', () => {
    mockConfigured('https://acme.service-now.com/');
    renderTab();

    fireEvent.click(screen.getByTestId('servicenow-renew'));

    expect(screen.getByTestId('servicenow-secret-sn_api_key')).toHaveValue('');
    expect(screen.getByTestId('servicenow-endpoint')).toHaveValue(
      'https://acme.service-now.com/'
    );

    fireEvent.click(screen.getByTestId('servicenow-cancel'));
    expect(screen.getByTestId('servicenow-connected')).toBeInTheDocument();
  });

  it('offers no renewal cancel while nothing is stored', () => {
    renderTab();

    expect(screen.queryByTestId('servicenow-cancel')).not.toBeInTheDocument();
  });
});

describe('ServiceNowConnection — saving', () => {
  it('writes the whole key in one PATCH carrying exactly the declared names', async () => {
    renderTab();

    type('servicenow-endpoint', 'https://acme.service-now.com/');
    fillCredentials();
    await submit();

    await waitFor(() => expect(patchMutation).toHaveBeenCalledTimes(1));
    expect(patchMutation).toHaveBeenCalledWith({
      settingClass: 'SEPSettings',
      key: 'DIAGNOSTICS_DELIVERY_INPUTS',
      value: {
        endpoint: 'https://acme.service-now.com',
        secrets: { sn_api_key: 'key-1', client_token: 'token-1' },
      },
    });
  });

  it('submits a declared name that form paths cannot express, verbatim', async () => {
    mockList({ data: sepGroups(['sn.api.key']) } as Partial<
      ReturnType<typeof useSettingsList>
    >);

    renderTab();

    type('servicenow-secret-sn.api.key', 'key-1');
    await submit();

    await waitFor(() => expect(patchMutation).toHaveBeenCalledTimes(1));
    expect(patchMutation.mock.calls[0][0].value).toEqual({
      secrets: { 'sn.api.key': 'key-1' },
    });
  });

  it('holds the submit until every declared credential is supplied', async () => {
    renderTab();

    type('servicenow-secret-sn_api_key', 'key-1');

    await waitFor(() =>
      expect(screen.getByTestId('servicenow-submit')).toBeDisabled()
    );

    type('servicenow-secret-client_token', 'token-1');

    await waitFor(() =>
      expect(screen.getByTestId('servicenow-submit')).toBeEnabled()
    );
  });

  it('refuses an endpoint that is not a URL before it reaches SEP', async () => {
    renderTab();

    fillCredentials();
    type('servicenow-endpoint', 'not-a-url');

    await waitFor(() =>
      expect(screen.getByTestId('servicenow-submit')).toBeDisabled()
    );
    expect(patchMutation).not.toHaveBeenCalled();
  });

  it('surfaces the 422 SEP answers with instead of swallowing it', () => {
    mockPatch(
      new ApiError({
        kind: 'http',
        status: 422,
        message: 'HTTP 422',
        data: {
          detail: [
            {
              loc: ['body', 'DIAGNOSTICS_DELIVERY_INPUTS'],
              msg: 'undeclared secret names: extra',
              type: 'value_error',
            },
          ],
        },
      })
    );

    renderTab();

    expect(screen.getByTestId('servicenow-save-error')).toHaveTextContent(
      'undeclared secret names: extra'
    );
  });

  it('keeps the typed credentials on screen when the save is rejected', async () => {
    patchMutation.mockRejectedValue(
      new ApiError({ kind: 'network', message: 'down' })
    );
    renderTab();

    fillCredentials();
    await submit();

    await waitFor(() => expect(patchMutation).toHaveBeenCalled());
    expect(screen.getByTestId('servicenow-secret-sn_api_key')).toHaveValue(
      'key-1'
    );
  });
});

describe('ServiceNowConnection — disconnecting', () => {
  it('is not offered while nothing is stored', () => {
    renderTab();

    expect(
      screen.queryByTestId('servicenow-disconnect')
    ).not.toBeInTheDocument();
  });

  it('confirms before clearing the stored inputs', async () => {
    mockConfigured();
    renderTab();

    fireEvent.click(screen.getByTestId('servicenow-disconnect'));
    expect(resetMutation).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId('servicenow-disconnect-confirm'));

    await waitFor(() =>
      expect(resetMutation).toHaveBeenCalledWith({
        settingClass: 'SEPSettings',
        key: 'DIAGNOSTICS_DELIVERY_INPUTS',
      })
    );
  });

  it('leaves the configuration alone when the confirmation is dismissed', () => {
    mockConfigured();
    renderTab();

    fireEvent.click(screen.getByTestId('servicenow-disconnect'));
    fireEvent.click(screen.getByTestId('servicenow-disconnect-cancel'));

    expect(resetMutation).not.toHaveBeenCalled();
  });
});

describe('ServiceNowConnection — testing the connection', () => {
  it('is offered on the connection it can actually probe, not on the form', () => {
    renderTab();
    expect(screen.queryByTestId('servicenow-test')).not.toBeInTheDocument();

    mockConfigured();
    renderTab();
    expect(screen.getByTestId('servicenow-test')).toBeInTheDocument();
  });

  it('probes delivery alone, and saves nothing while doing it', () => {
    mockConfigured('https://acme.service-now.com');
    renderTab();

    fireEvent.click(screen.getByTestId('servicenow-test'));

    expect(probeMutation).toHaveBeenCalledWith({ targets: ['delivery'] });
    expect(patchMutation).not.toHaveBeenCalled();
    expect(resetMutation).not.toHaveBeenCalled();
  });

  it('verifies on its own once the form has just saved', async () => {
    // The save invalidates the settings query, so the next render of the tab
    // reads the credentials it just stored and lands on the connected screen.
    patchMutation.mockImplementation(async () => {
      mockConfigured();
    });
    renderTab();

    fillCredentials();
    await submit();

    await waitFor(() =>
      expect(screen.getByTestId('servicenow-connected')).toBeInTheDocument()
    );
    expect(probeMutation).toHaveBeenCalledWith({ targets: ['delivery'] });
  });

  it('does not re-probe when a stored connection is merely opened', () => {
    mockConfigured();
    renderTab();

    expect(probeMutation).not.toHaveBeenCalled();
  });

  it('stays legible while the probe is still running', () => {
    mockConfigured();
    mockProbe({ isPending: true });
    renderTab();

    const button = screen.getByTestId('servicenow-test');
    expect(button).toBeDisabled();
    expect(button).toHaveTextContent(Messages.serviceNow.test.testing);
  });

  it('withholds the previous verdict while a re-test is in flight', () => {
    mockConfigured();
    mockProbe({ isPending: true, data: [probed('reachable')] });
    renderTab();

    expect(
      screen.queryByTestId('servicenow-test-result')
    ).not.toBeInTheDocument();
  });

  it('drops a verdict once the configuration it described has changed', () => {
    mockConfigured('https://acme.service-now.com');
    mockProbe({ data: [probed('reachable')] });
    const { rerender } = renderTab();

    expect(screen.getByTestId('servicenow-test-result')).toBeInTheDocument();
    expect(probeReset).not.toHaveBeenCalled();

    // What another administrator saving from a second session looks like here.
    mockConfigured('https://other.service-now.com');
    rerender(
      <TestWrapper>
        {wrapWithSnackbarProvider(<ServiceNowConnection />)}
      </TestWrapper>
    );

    expect(probeReset).toHaveBeenCalledTimes(1);
  });

  it.each([
    'reachable',
    'auth_failed',
    'probe_undeclared',
  ] as ConnectivityStatus[])(
    'reports the %s verdict in its own words',
    (status) => {
      mockConfigured();
      mockProbe({ data: [probed(status)] });
      renderTab();

      expect(screen.getByTestId('servicenow-test-result')).toHaveTextContent(
        Messages.serviceNow.test.statuses[status]
      );
      expect(
        screen.queryByTestId('servicenow-test-error')
      ).not.toBeInTheDocument();
    }
  );

  it('reports a probe that never ran as that, not as a bad connection', () => {
    mockConfigured();
    mockProbe({
      error: new ApiError({ kind: 'http', status: 403, message: 'HTTP 403' }),
    });
    renderTab();

    expect(screen.getByTestId('servicenow-test-error')).toHaveTextContent(
      Messages.serviceNow.errors.forbidden
    );
    expect(
      screen.queryByTestId('servicenow-test-result')
    ).not.toBeInTheDocument();
  });

  it('leaves the connection itself standing whatever the verdict says', () => {
    mockConfigured('https://acme.service-now.com');
    mockProbe({ data: [probed('unreachable')] });
    renderTab();

    expect(screen.getByTestId('servicenow-connected')).toHaveTextContent(
      Messages.serviceNow.connectedTitle
    );
    expect(
      screen.getByTestId('servicenow-connected-endpoint')
    ).toHaveTextContent('https://acme.service-now.com');
    expect(screen.getByTestId('servicenow-test-result')).toHaveTextContent(
      Messages.serviceNow.test.statuses.unreachable
    );
  });
});
