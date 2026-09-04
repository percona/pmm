import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import {
  ApiError,
  REDACTED_SECRET,
  SettingClassGroup,
  useResetSetting,
  usePatchSetting,
  useSettingsList,
} from '@sep/api';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithSnackbarProvider } from 'utils/testUtils';
import { Messages } from '../../Settings.messages';
import { ServiceNowConnectionForm } from './ServiceNowConnectionForm';

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  useSettingsList: vi.fn(),
  usePatchSetting: vi.fn(),
  useResetSetting: vi.fn(),
}));

const settingsList = vi.mocked(useSettingsList);
const patchSetting = vi.mocked(usePatchSetting);
const resetSetting = vi.mocked(useResetSetting);

const patchMutation = vi.fn();
const resetMutation = vi.fn();

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
    error: null,
    ...overrides,
  } as ReturnType<typeof useSettingsList>);
};

const mockPatch = (error: ApiError | null = null) => {
  patchSetting.mockReturnValue({
    mutateAsync: patchMutation,
    error,
  } as unknown as ReturnType<typeof usePatchSetting>);
};

const renderForm = () =>
  render(
    <TestWrapper>
      {wrapWithSnackbarProvider(<ServiceNowConnectionForm />)}
    </TestWrapper>
  );

const type = (testId: string, value: string) =>
  fireEvent.change(screen.getByTestId(testId), { target: { value } });

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
  resetSetting.mockReturnValue({
    mutateAsync: resetMutation,
    isPending: false,
  } as unknown as ReturnType<typeof useResetSetting>);
});

describe('ServiceNowConnectionForm — rendering', () => {
  it('renders one field per declared secret name', () => {
    renderForm();

    expect(screen.getByTestId('servicenow-secret-sn_api_key')).toHaveAttribute(
      'type',
      'password'
    );
    expect(
      screen.getByTestId('servicenow-secret-client_token')
    ).toBeInTheDocument();
  });

  it('follows the plan when an image renames a declared secret', () => {
    settingsList.mockReturnValue({
      data: sepGroups(['sn_api_key', 'renamed_token'], {
        sn_api_key: REDACTED_SECRET,
        client_token: REDACTED_SECRET,
      }),
      isLoading: false,
      error: null,
    } as ReturnType<typeof useSettingsList>);

    renderForm();

    expect(
      screen.getByTestId('servicenow-secret-renamed_token')
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId('servicenow-secret-client_token')
    ).not.toBeInTheDocument();
    expect(screen.getByTestId('servicenow-status')).toHaveTextContent(
      Messages.serviceNow.status.drifted
    );
  });

  it('submits a declared name that form paths cannot express, verbatim', async () => {
    settingsList.mockReturnValue({
      data: sepGroups(['sn.api.key']),
      isLoading: false,
      error: null,
    } as ReturnType<typeof useSettingsList>);

    renderForm();

    type('servicenow-secret-sn.api.key', 'key-1');
    await submit();

    await waitFor(() => expect(patchMutation).toHaveBeenCalledTimes(1));
    expect(patchMutation.mock.calls[0][0].value).toEqual({
      secrets: { 'sn.api.key': 'key-1' },
    });
  });

  it('reports an unsaved connection as not configured rather than as an error', () => {
    renderForm();

    const status = screen.getByTestId('servicenow-status');
    expect(status).toHaveTextContent(Messages.serviceNow.status.notConfigured);
    expect(status).toHaveClass('MuiAlert-colorInfo');
  });

  it('reports a saved-but-empty secret as not configured', () => {
    settingsList.mockReturnValue({
      data: sepGroups(['sn_api_key', 'client_token'], {
        sn_api_key: REDACTED_SECRET,
        client_token: '',
      }),
      isLoading: false,
      error: null,
    } as ReturnType<typeof useSettingsList>);

    renderForm();

    expect(screen.getByTestId('servicenow-status')).toHaveTextContent(
      Messages.serviceNow.status.notConfigured
    );
  });

  it('shows the stored values back, masked', () => {
    settingsList.mockReturnValue({
      data: sepGroups(
        ['sn_api_key', 'client_token'],
        { sn_api_key: REDACTED_SECRET, client_token: REDACTED_SECRET },
        'https://acme.service-now.com/'
      ),
      isLoading: false,
      error: null,
    } as ReturnType<typeof useSettingsList>);

    renderForm();

    expect(screen.getByTestId('servicenow-endpoint')).toHaveValue(
      'https://acme.service-now.com/'
    );
    expect(screen.getByTestId('servicenow-secret-sn_api_key')).toHaveValue(
      REDACTED_SECRET
    );
    expect(screen.getByTestId('servicenow-status')).toHaveTextContent(
      Messages.serviceNow.status.configured
    );
  });

  it('shows a spinner while the settings load', () => {
    mockList({ isLoading: true, data: undefined });
    renderForm();

    expect(screen.getByTestId('servicenow-loading')).toBeInTheDocument();
  });

  it('explains a load that was refused rather than showing an empty form', () => {
    mockList({
      data: undefined,
      error: new ApiError({ kind: 'http', status: 403, message: 'HTTP 403' }),
    });
    renderForm();

    expect(screen.getByTestId('servicenow-load-error')).toHaveTextContent(
      Messages.serviceNow.errors.forbidden
    );
    expect(screen.queryByTestId('servicenow-submit')).not.toBeInTheDocument();
  });

  it('still offers the endpoint when the plan declares no credentials', () => {
    settingsList.mockReturnValue({
      data: sepGroups([]),
      isLoading: false,
      error: null,
    } as ReturnType<typeof useSettingsList>);

    renderForm();

    expect(screen.getByTestId('servicenow-no-secrets')).toBeInTheDocument();
    expect(screen.getByTestId('servicenow-endpoint')).toBeInTheDocument();
  });

  it('offers nothing when the deployment does not carry the key at all', () => {
    settingsList.mockReturnValue({
      data: [
        { setting_class: 'SEPSettings', is_app_owned: false, settings: [] },
      ] as SettingClassGroup[],
      isLoading: false,
      error: null,
    } as ReturnType<typeof useSettingsList>);

    renderForm();

    expect(screen.getByTestId('servicenow-unavailable')).toHaveTextContent(
      Messages.serviceNow.unavailable
    );
    expect(screen.queryByTestId('servicenow-endpoint')).not.toBeInTheDocument();
  });
});

describe('ServiceNowConnectionForm — saving', () => {
  it('writes the whole key in one PATCH carrying exactly the declared names', async () => {
    renderForm();

    type('servicenow-endpoint', 'https://acme.service-now.com/');
    type('servicenow-secret-sn_api_key', 'key-1');
    type('servicenow-secret-client_token', 'token-1');
    await submit();

    await waitFor(() => expect(patchMutation).toHaveBeenCalledTimes(1));
    expect(patchMutation).toHaveBeenCalledWith({
      settingClass: 'SEPSettings',
      key: 'DIAGNOSTICS_DELIVERY_INPUTS',
      value: {
        endpoint: 'https://acme.service-now.com/',
        secrets: { sn_api_key: 'key-1', client_token: 'token-1' },
      },
    });
  });

  it('keeps a stored secret by resubmitting the mask it was shown', async () => {
    settingsList.mockReturnValue({
      data: sepGroups(
        ['sn_api_key', 'client_token'],
        { sn_api_key: REDACTED_SECRET, client_token: REDACTED_SECRET },
        'https://acme.service-now.com/'
      ),
      isLoading: false,
      error: null,
    } as ReturnType<typeof useSettingsList>);

    renderForm();

    type('servicenow-endpoint', 'https://other.service-now.com/');
    await submit();

    await waitFor(() => expect(patchMutation).toHaveBeenCalledTimes(1));
    expect(patchMutation.mock.calls[0][0].value).toEqual({
      endpoint: 'https://other.service-now.com/',
      secrets: {
        sn_api_key: REDACTED_SECRET,
        client_token: REDACTED_SECRET,
      },
    });
  });

  it('accepts clearing a secret as an explicit unconfigured save', async () => {
    settingsList.mockReturnValue({
      data: sepGroups(['sn_api_key', 'client_token'], {
        sn_api_key: REDACTED_SECRET,
        client_token: REDACTED_SECRET,
      }),
      isLoading: false,
      error: null,
    } as ReturnType<typeof useSettingsList>);

    renderForm();

    type('servicenow-secret-sn_api_key', '');
    await submit();

    await waitFor(() => expect(patchMutation).toHaveBeenCalledTimes(1));
    expect(patchMutation.mock.calls[0][0].value).toEqual({
      secrets: { sn_api_key: '', client_token: REDACTED_SECRET },
    });
  });

  it('refuses an endpoint that is not a URL before it reaches SEP', async () => {
    renderForm();

    type('servicenow-endpoint', 'not-a-url');
    type('servicenow-secret-sn_api_key', 'key-1');

    await waitFor(() =>
      expect(screen.getByTestId('servicenow-submit')).toBeDisabled()
    );
    expect(patchMutation).not.toHaveBeenCalled();
  });

  it('surfaces the 422 SEP answers with instead of swallowing it', async () => {
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

    renderForm();

    expect(screen.getByTestId('servicenow-save-error')).toHaveTextContent(
      'undeclared secret names: extra'
    );
  });

  it('keeps the form on screen when the save is rejected', async () => {
    patchMutation.mockRejectedValue(
      new ApiError({ kind: 'network', message: 'down' })
    );
    renderForm();

    type('servicenow-secret-sn_api_key', 'key-1');
    await submit();

    await waitFor(() => expect(patchMutation).toHaveBeenCalled());
    expect(screen.getByTestId('servicenow-secret-sn_api_key')).toHaveValue(
      'key-1'
    );
  });
});

describe('ServiceNowConnectionForm — disconnecting', () => {
  const renderConfigured = () => {
    settingsList.mockReturnValue({
      data: sepGroups(['sn_api_key', 'client_token'], {
        sn_api_key: REDACTED_SECRET,
        client_token: REDACTED_SECRET,
      }),
      isLoading: false,
      error: null,
    } as ReturnType<typeof useSettingsList>);
    return renderForm();
  };

  it('is not offered while nothing is stored', () => {
    renderForm();

    expect(
      screen.queryByTestId('servicenow-disconnect')
    ).not.toBeInTheDocument();
  });

  it('confirms before clearing the stored inputs', async () => {
    renderConfigured();

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
    renderConfigured();

    fireEvent.click(screen.getByTestId('servicenow-disconnect'));
    fireEvent.click(screen.getByTestId('servicenow-disconnect-cancel'));

    expect(resetMutation).not.toHaveBeenCalled();
  });
});
