import { render, screen } from '@testing-library/react';
import {
  ApiError,
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

const renderGate = () =>
  render(
    <ServiceNowSetupGate>
      <div data-testid="atw-app" />
    </ServiceNowSetupGate>,
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
