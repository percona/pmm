import type { AxiosRequestConfig } from 'axios';
import {
  FrontendSettings,
  ReadonlySettings,
  Settings,
  UpdateSettingsPayload,
} from 'types/settings.types';

export const READONLY_SETTINGS_MOCK: ReadonlySettings = {
  updatesEnabled: true,
  telemetryEnabled: false,
  advisorEnabled: true,
  alertingEnabled: true,
  pmmPublicAddress: '',
  backupManagementEnabled: true,
  azurediscoverEnabled: false,
  enableAccessControl: false,
  omEnabled: false,
};

export const SETTINGS_MOCK: Settings = {
  ...READONLY_SETTINGS_MOCK,
  metricsResolutions: {
    hr: '5s',
    mr: '10s',
    lr: '60s',
  },
  dataRetention: '2592000s',
  awsPartitions: ['aws'],
  advisorRunIntervals: {
    rareInterval: '280800s',
    standardInterval: '86400s',
    frequentInterval: '14400s',
  },
  enableInternalPgQan: false,
  defaultRoleId: 1,
};

export const FRONTEND_SETTINGS_MOCK: FrontendSettings = {
  anonymousEnabled: false,
  appSubUrl: '',
  apps: {},
  buildInfo: {
    version: '',
    versionString: '',
  },
  exploreEnabled: true,
  featureToggles: {
    exploreMetrics: true,
  },
  unifiedAlertingEnabled: true,
  disableLoginForm: false,
  auth: {
    disableLogin: false,
  },
};

export const getSettings = vi.fn<
  (config?: AxiosRequestConfig) => Promise<Settings>
>(async () => SETTINGS_MOCK);

export const getReadonlySettings = vi.fn(
  async (): Promise<ReadonlySettings> => READONLY_SETTINGS_MOCK
);

export const getFrontendSettings = vi.fn(
  async (): Promise<FrontendSettings> => FRONTEND_SETTINGS_MOCK
);

export const updateSettings = vi.fn<
  (payload: UpdateSettingsPayload) => Promise<Settings>
>(async () => SETTINGS_MOCK);
