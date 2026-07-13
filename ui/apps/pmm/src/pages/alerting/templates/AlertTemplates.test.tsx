import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import * as templatesApi from 'api/alert-templates';
import {
  Severity,
  Template,
  TemplateSource,
} from 'types/alert-templates.types';
import { AlertTemplates } from './AlertTemplates';

vi.mock('api/alert-templates');

const listTemplatesMock = vi.mocked(templatesApi.listTemplates);

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
});

const renderPage = () =>
  render(
    <TestWrapper>
      {wrapWithQueryProvider(wrapWithSnackbarProvider(<AlertTemplates />))}
    </TestWrapper>
  );

describe('AlertTemplates', () => {
  beforeEach(() => {
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
});
