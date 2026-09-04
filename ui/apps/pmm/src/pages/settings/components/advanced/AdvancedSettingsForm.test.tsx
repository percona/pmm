import { render, screen } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import {
  LockReason,
  SettingName,
  type SettingLock,
  type Settings as SettingsType,
} from 'types/settings.types';
import { AdvancedSettingsForm } from './AdvancedSettingsForm';

vi.mock('api/settings');

const settingsWith = (lockedSettings?: SettingLock[]): SettingsType =>
  ({
    dataRetention: '2592000s',
    lockedSettings,
  }) as SettingsType;

const renderForm = (lockedSettings?: SettingLock[]) =>
  render(
    <TestWrapper>
      {wrapWithQueryProvider(
        <AdvancedSettingsForm settings={settingsWith(lockedSettings)} />
      )}
    </TestWrapper>
  );

const retentionInput = () => screen.getByTestId('retention-number-input');

describe('AdvancedSettingsForm data retention lock', () => {
  it('is editable when the server reports no lock', () => {
    renderForm();

    expect(retentionInput()).toBeEnabled();
  });

  it('is editable when the server locks some other setting', () => {
    renderForm([
      {
        setting: SettingName.telemetryEnabled,
        reason: LockReason.environment,
        environmentVariable: 'PMM_ENABLE_TELEMETRY',
      },
    ]);

    expect(retentionInput()).toBeEnabled();
  });

  it('is disabled and points at the chart when high availability locks it', () => {
    renderForm([
      {
        setting: SettingName.dataRetention,
        reason: LockReason.highAvailability,
      },
    ]);

    expect(retentionInput()).toBeDisabled();
    expect(screen.getByText(/dataRetentionDays/)).toBeInTheDocument();
  });

  it('is disabled and names the variable when the environment locks it', () => {
    renderForm([
      {
        setting: SettingName.dataRetention,
        reason: LockReason.environment,
        environmentVariable: 'PMM_DATA_RETENTION',
      },
    ]);

    expect(retentionInput()).toBeDisabled();
    expect(screen.getByText(/PMM_DATA_RETENTION/)).toBeInTheDocument();
  });

  // The server sets environmentVariable only for an environment lock, so a malformed response
  // must still produce a usable message rather than "Set by the undefined variable".
  it('falls back to a named variable when the server omits it', () => {
    renderForm([
      {
        setting: SettingName.dataRetention,
        reason: LockReason.environment,
      },
    ]);

    expect(retentionInput()).toBeDisabled();
    expect(screen.getByText(/PMM_DATA_RETENTION/)).toBeInTheDocument();
  });
});
