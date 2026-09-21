import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import * as templatesApi from 'api/alert-templates';
import * as fileUtils from 'utils/file.utils';
import {
  Severity,
  Template,
  TemplateCategory,
  TemplateSource,
} from 'types/alert-templates.types';
import { AlertTemplates } from './AlertTemplates';

vi.mock('api/alert-templates');
vi.mock('utils/file.utils');
// The real modal renders percona-ui's Dialog + multiline TextInput, which
// isn't renderable under jsdom; stub it to verify the props it receives.
vi.mock('./modal-create-template', () => ({
  CreateTemplateModal: ({
    open,
    initialYaml,
  }: {
    open: boolean;
    initialYaml?: string;
  }) =>
    open ? (
      <div
        data-testid="create-template-modal-stub"
        data-initial-yaml={initialYaml ?? ''}
      />
    ) : null,
}));

const listTemplatesMock = vi.mocked(templatesApi.listTemplates);
const downloadTextFileMock = vi.mocked(fileUtils.downloadTextFile);

const makeTemplate = (
  name: string,
  source: TemplateSource,
  summary: string
): Template => ({
  name,
  summary,
  expr: 'up == 0',
  params: [],
  for: '60s',
  severity: Severity.WARNING,
  labels: {},
  annotations: {},
  source,
  yaml: `# ${name}\nexpr: up == 0\n`,
  category: TemplateCategory.UNSPECIFIED,
});

const renderPage = () =>
  render(
    <TestWrapper>
      {wrapWithQueryProvider(wrapWithSnackbarProvider(<AlertTemplates />))}
    </TestWrapper>
  );

describe('AlertTemplates', () => {
  beforeEach(() => {
    downloadTextFileMock.mockClear();
    listTemplatesMock.mockResolvedValue({
      totalItems: 2,
      totalPages: 1,
      templates: [
        makeTemplate('builtin_one', TemplateSource.BUILT_IN, 'Built-in one'),
        makeTemplate('user_one', TemplateSource.USER_API, 'User one'),
      ],
    });
  });

  it('renders template rows', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByText('Built-in one')).toBeInTheDocument()
    );
    expect(screen.getByText('User one')).toBeInTheDocument();
  });

  it('shows the add button for admins', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('add-alert-template')).toBeInTheDocument()
    );
  });

  it('offers edit/delete in the row menu only for user-created templates', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getAllByTestId('template-actions-menu')).toHaveLength(2)
    );
    // Row order matches data order: built-in first, user-created second.
    const [builtInMenu, userMenu] = screen.getAllByTestId(
      'template-actions-menu'
    );

    // Built-in: Create alert rule + View (ungated), no edit/delete.
    fireEvent.click(builtInMenu);
    await screen.findByTestId('create-alert-rule');
    expect(screen.getByTestId('view-alert-template')).toBeInTheDocument();
    expect(screen.queryByTestId('edit-alert-template')).toBeNull();
    expect(screen.queryByTestId('delete-alert-template')).toBeNull();
    fireEvent.keyDown(screen.getByTestId('create-alert-rule'), {
      key: 'Escape',
    });

    // User-created: edit and delete are available.
    fireEvent.click(userMenu);
    await screen.findByTestId('edit-alert-template');
    expect(screen.getByTestId('delete-alert-template')).toBeInTheDocument();
  });

  it('opens the view modal from the row menu', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getAllByTestId('template-actions-menu').length).toBe(2)
    );
    fireEvent.click(screen.getAllByTestId('template-actions-menu')[0]);
    fireEvent.click(await screen.findByTestId('view-alert-template'));

    const modal = await screen.findByTestId('view-template-modal');
    expect(modal).toBeInTheDocument();
    expect(screen.getByTestId('view-template-yaml')).toHaveValue(
      '# builtin_one\nexpr: up == 0\n'
    );
  });

  it('copies the template YAML from the row menu', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getAllByTestId('template-actions-menu').length).toBe(2)
    );
    fireEvent.click(screen.getAllByTestId('template-actions-menu')[0]);
    fireEvent.click(await screen.findByTestId('copy-alert-template'));

    await waitFor(() =>
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
        '# builtin_one\nexpr: up == 0\n'
      )
    );
  });

  it('exports the template YAML as a file from the row menu', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getAllByTestId('template-actions-menu').length).toBe(2)
    );
    fireEvent.click(screen.getAllByTestId('template-actions-menu')[0]);
    fireEvent.click(await screen.findByTestId('export-alert-template'));

    expect(downloadTextFileMock).toHaveBeenCalledWith(
      'builtin_one.yaml',
      '# builtin_one\nexpr: up == 0\n'
    );
  });

  it('offers duplicate in the row menu only for admins', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getAllByTestId('template-actions-menu').length).toBe(2)
    );
    fireEvent.click(screen.getAllByTestId('template-actions-menu')[0]);
    expect(
      await screen.findByTestId('duplicate-alert-template')
    ).toBeInTheDocument();
  });

  it('opens the create modal pre-filled when duplicating a template', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getAllByTestId('template-actions-menu').length).toBe(2)
    );
    fireEvent.click(screen.getAllByTestId('template-actions-menu')[0]);
    fireEvent.click(await screen.findByTestId('duplicate-alert-template'));

    const modal = await screen.findByTestId('create-template-modal-stub');
    expect(modal).toHaveAttribute(
      'data-initial-yaml',
      '# builtin_one\nexpr: up == 0\n'
    );
  });

  it('shows a bulk export button once rows are selected and exports each one', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getAllByTestId('template-actions-menu').length).toBe(2)
    );
    expect(screen.queryByTestId('export-selected-templates')).toBeNull();

    const checkboxes = screen.getAllByRole('checkbox', {
      name: /toggle select row/i,
    });
    fireEvent.click(checkboxes[0]);
    fireEvent.click(checkboxes[1]);

    const exportButton = await screen.findByTestId('export-selected-templates');
    fireEvent.click(exportButton);

    expect(downloadTextFileMock).toHaveBeenCalledWith(
      'builtin_one.yaml',
      '# builtin_one\nexpr: up == 0\n'
    );
    expect(downloadTextFileMock).toHaveBeenCalledWith(
      'user_one.yaml',
      '# user_one\nexpr: up == 0\n'
    );
    expect(downloadTextFileMock).toHaveBeenCalledTimes(2);
  });
});
