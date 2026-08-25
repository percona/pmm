import { render, screen } from '@testing-library/react';
import {
  ApiError,
  AuthContext,
  REDACTED_SECRET,
  SettingClassGroup,
  useSettingsList,
} from '@sep/api';
import { TestWrapper } from 'utils/testWrapper';
import { PMM_SERVICENOW_SETTINGS_PATH } from 'lib/constants';
import { ServiceNowSetupGate } from './ServiceNowSetupGate';
import { Messages } from './ServiceNowSetupGate.messages';

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  useSettingsList: vi.fn(),
}));

const settingsList = vi.mocked(useSettingsList);

const setting = (key: string, value: unknown, hasOverride = false) =>
  ({
    key,
    value,
    has_override: hasOverride,
    setting_class: 'SEPSettings',
    type: 'object',
  }) as unknown as SettingClassGroup['settings'][number];

const sepGroups = (
  declared: string[],
  storedSecrets?: Record<string, string>
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
        { endpoint: '', secrets: storedSecrets ?? {} },
        storedSecrets !== undefined
      ),
    ],
  },
];

const mockList = (
  overrides: Partial<ReturnType<typeof useSettingsList>> = {}
) => {
  settingsList.mockReturnValue({
    data: undefined,
    isLoading: false,
    error: null,
    ...overrides,
  } as ReturnType<typeof useSettingsList>);
};

/**
 * Defaults to an administrator: the prompt only ever renders for one, so every
 * case below except the read-only ones is an admin case.
 */
const renderGate = (isAdmin = true) =>
  render(
    <AuthContext.Provider value={{ isAdmin }}>
      <ServiceNowSetupGate>
        <div data-testid="atw-app" />
      </ServiceNowSetupGate>
    </AuthContext.Provider>,
    { wrapper: TestWrapper }
  );

describe('ServiceNowSetupGate', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the app once the connection is configured', () => {
    mockList({ data: sepGroups(['sn_api_key'], { sn_api_key: 'secret' }) });
    renderGate();

    expect(screen.getByTestId('atw-app')).toBeInTheDocument();
    expect(
      screen.queryByTestId('servicenow-setup-prompt')
    ).not.toBeInTheDocument();
  });

  it('prompts for setup when nothing is stored', () => {
    mockList({ data: sepGroups(['sn_api_key']) });
    renderGate();

    expect(screen.getByTestId('servicenow-setup-prompt')).toBeInTheDocument();
    expect(screen.getByText(Messages.title)).toBeInTheDocument();
    expect(screen.queryByTestId('atw-app')).not.toBeInTheDocument();
  });

  it('prompts for setup when the stored values no longer match the plan', () => {
    mockList({ data: sepGroups(['sn_token'], { sn_api_key: 'secret' }) });
    renderGate();

    expect(screen.getByTestId('servicenow-setup-prompt')).toBeInTheDocument();
  });

  it('links the call to action at the ServiceNow settings tab', () => {
    mockList({ data: sepGroups(['sn_api_key']) });
    renderGate();

    expect(screen.getByTestId('servicenow-setup-cta')).toHaveAttribute(
      'href',
      PMM_SERVICENOW_SETTINGS_PATH
    );
  });

  it('renders the app when SEP does not carry the delivery inputs key', () => {
    mockList({
      data: [
        {
          setting_class: 'SEPSettings',
          is_app_owned: false,
          settings: [setting('DIAGNOSTICS_DELIVERY', { secrets: {} })],
        },
      ] as SettingClassGroup[],
    });
    renderGate();

    expect(screen.getByTestId('atw-app')).toBeInTheDocument();
    expect(
      screen.queryByTestId('servicenow-setup-prompt')
    ).not.toBeInTheDocument();
  });

  it('renders the app when the settings read failed', () => {
    mockList({
      error: new ApiError({ kind: 'network', message: 'down' }),
    });
    renderGate();

    expect(screen.getByTestId('atw-app')).toBeInTheDocument();
  });

  it('waits while the settings are loading', () => {
    mockList({ isLoading: true });
    renderGate();

    expect(screen.getByLabelText(Messages.loading)).toBeInTheDocument();
    expect(screen.queryByTestId('atw-app')).not.toBeInTheDocument();
  });
});

describe('ServiceNowSetupGate — read-only sessions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the app for a non-admin and reads no settings', () => {
    // SEP holds GET /sep/admin/settings to administrators, reads included, so
    // the request is skipped rather than fired to be refused.
    mockList({ data: sepGroups(['sn_api_key']) });
    renderGate(false);

    expect(screen.getByTestId('atw-app')).toBeInTheDocument();
    expect(settingsList).toHaveBeenCalledWith({ enabled: false });
  });

  it('shows a non-admin the app rather than a prompt they cannot act on', () => {
    // Same unconfigured deployment that prompts an admin: the prompt's only
    // call to action is a settings tab a non-admin cannot open.
    mockList({ data: sepGroups(['sn_token'], { sn_api_key: 'secret' }) });
    renderGate(false);

    expect(screen.getByTestId('atw-app')).toBeInTheDocument();
    expect(
      screen.queryByTestId('servicenow-setup-prompt')
    ).not.toBeInTheDocument();
  });
});
