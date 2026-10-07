import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { SETTINGS_MOCK } from 'api/__mocks__/settings';
import { Settings } from 'types/settings.types';
import { Messages } from '../../Settings.messages';
import { AdvancedSettingsForm } from './AdvancedSettingsForm';

const enqueueSnackbar = vi.fn();
vi.mock('notistack', () => ({
  enqueueSnackbar: (...args: unknown[]) => enqueueSnackbar(...args),
}));

// The real mutation would need a server. Only `onSuccess` matters here: what the
// form says once a save has landed.
const mutateAsync = vi.fn(
  async (
    _payload: unknown,
    options?: { onSuccess?: () => void; onError?: (error: Error) => void }
  ) => {
    options?.onSuccess?.();
  }
);
vi.mock('hooks/api/useSettings', () => ({
  useUpdateSettings: () => ({ mutateAsync }),
}));

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

  // "Settings updated" names neither what was switched on nor where it went, which
  // is the whole of the design review's first-run complaint (P9). Turning it on is
  // the one moment the reader needs both.
  describe('feedback once the save lands', () => {
    beforeEach(() => {
      enqueueSnackbar.mockClear();
      mutateAsync.mockClear();
    });

    // The real button, not a synthetic submit on the form: it is disabled until
    // the form is dirty, so submitting the element directly would exercise a path
    // no user can reach. The wait is not incidental either - the form is
    // `mode: 'onChange'`, so `isDirty` and `isValid` land a tick after the change
    // event, and a click before that hits a disabled button and does nothing.
    const apply = async () => {
      const button = screen.getByRole('button', {
        name: Messages.applyChanges,
      });
      await waitFor(() => expect(button).toBeEnabled());
      fireEvent.click(button);
    };

    it('names the feature and links to it when it is switched on', async () => {
      renderForm({ extensionsEnabled: true, omEnabled: false });

      fireEvent.click(operationsSwitch());
      await apply();

      await waitFor(() => expect(enqueueSnackbar).toHaveBeenCalled());

      // The message is a node, not a string, so it is rendered to be read -
      // asserting on the element tree would test JSX shape rather than what a
      // user sees.
      const [message] = enqueueSnackbar.mock.calls[0];
      const { getByRole, getByText } = render(<>{message}</>);

      getByText(m.openManagerEnabled, { exact: false });
      expect(
        getByRole('link', { name: m.openManagerEnabledAction })
      ).toHaveAttribute('href', '/pmm-ui/operations');
    });

    // Saving an unrelated field while it is already on must not tell someone
    // again where to find something they have been using. The public address is
    // dirtied only to make the form submittable.
    it('stays generic when it was already on', async () => {
      renderForm({ extensionsEnabled: true, omEnabled: true });

      // By test id: the field's label is a sibling node rather than a `for=`/`id=`
      // pair, so the input has no accessible name to query it by.
      fireEvent.change(screen.getByTestId('publicAddress-text-input'), {
        target: { value: 'pmm.example.com' },
      });
      await apply();

      await waitFor(() => expect(enqueueSnackbar).toHaveBeenCalled());
      expect(enqueueSnackbar).toHaveBeenCalledWith(Messages.service.success, {
        variant: 'success',
      });
    });

    // Turning it off is not an onboarding moment either.
    it('stays generic when it is switched off', async () => {
      renderForm({ extensionsEnabled: true, omEnabled: true });

      fireEvent.click(operationsSwitch());
      await apply();

      await waitFor(() => expect(enqueueSnackbar).toHaveBeenCalled());
      expect(enqueueSnackbar).toHaveBeenCalledWith(Messages.service.success, {
        variant: 'success',
      });
    });
  });
});
