import { fireEvent, render, screen, within } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { SETTINGS_MOCK } from 'api/__mocks__/settings';
import { Settings } from 'types/settings.types';
import { Messages } from '../../Settings.messages';
import { AdvancedSettingsForm } from './AdvancedSettingsForm';

const m = Messages.advanced;

// The control itself, not its label: `SwitchInput` renders the label through
// `FormControlLabel`, which is not a `for=`/`id=` pair a label-text query can follow.
// The input carries `role="switch"`, which overrides its implicit checkbox role.
const operationsSwitch = () =>
  within(screen.getByTestId('advanced-open-manager')).getByRole('switch');

const renderForm = (overrides: Partial<Settings> = {}) =>
  render(
    wrapWithQueryProvider(
      <TestWrapper>
        <AdvancedSettingsForm settings={{ ...SETTINGS_MOCK, ...overrides }} />
      </TestWrapper>
    )
  );

describe('AdvancedSettingsForm: Operations for MongoDB', () => {
  it('offers the switch under its own Developer preview section', () => {
    renderForm({ extensionsEnabled: true });

    expect(screen.getByText(m.developerPreviewLegend)).toBeInTheDocument();
    expect(screen.getByTestId('advanced-open-manager')).toBeInTheDocument();
    expect(screen.getByText(m.openManagerLabel)).toBeInTheDocument();
  });

  // The switch sits in the Developer preview section, so it must not also be a
  // child of the Technical preview one -- which is where it used to live, beside
  // Azure monitoring and Access control.
  it('is not part of the Technical preview section', () => {
    renderForm({ extensionsEnabled: true });

    const technicalPreview = screen
      .getByText(m.technicalPreviewLegend)
      .closest('div.MuiStack-root')?.parentElement;

    expect(technicalPreview).not.toBeNull();
    expect(
      technicalPreview?.contains(screen.getByTestId('advanced-open-manager'))
    ).toBe(false);
  });

  it('explains what it does without needing a hover', () => {
    renderForm({ extensionsEnabled: true });

    expect(screen.getByText(m.openManagerTooltip)).toBeInTheDocument();
  });

  // Enabling it without PMM Extensions is refused server-side (validateEnableOm),
  // so the precondition is stated before Apply rather than discovered after it.
  it('is disabled, with a reason, when PMM Extensions is off', () => {
    renderForm({ extensionsEnabled: false });

    expect(operationsSwitch()).toBeDisabled();
    expect(
      screen.getByTestId('advanced-open-manager-blocked')
    ).toHaveTextContent(m.openManagerRequiresExtensions);
  });

  // Switching it off needs nothing from Extensions, and validateEnableOm accepts it,
  // so an admin whose Extensions went away must still be able to turn it off here.
  it('stays enabled when it is already on, even with PMM Extensions off', () => {
    renderForm({ extensionsEnabled: false, omEnabled: true });

    expect(operationsSwitch()).toBeEnabled();
    expect(screen.queryByTestId('advanced-open-manager-blocked')).toBeNull();

    // Flipping it off in the form, before Apply, must not lock it there: the block
    // follows the saved setting, not the draft, so the flip can still be undone.
    fireEvent.click(operationsSwitch());

    expect(operationsSwitch()).not.toBeChecked();
    expect(operationsSwitch()).toBeEnabled();
  });

  it('is enabled, with no blocking notice, when PMM Extensions is on', () => {
    renderForm({ extensionsEnabled: true });

    expect(operationsSwitch()).toBeEnabled();
    expect(screen.queryByTestId('advanced-open-manager-blocked')).toBeNull();
  });
});
