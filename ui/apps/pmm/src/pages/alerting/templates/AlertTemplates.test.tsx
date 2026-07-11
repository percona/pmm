import { render, screen, waitFor } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
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
  yaml: '',
});

const renderPage = () =>
  render(
    <TestWrapper>{wrapWithQueryProvider(<AlertTemplates />)}</TestWrapper>
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

  it('enables edit/delete only for user-created templates', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getAllByTestId('edit-alert-template')).toHaveLength(2)
    );
    const editButtons = screen.getAllByTestId('edit-alert-template');
    // Row order matches data order: built-in first (disabled), user second.
    expect(editButtons[0]).toBeDisabled();
    expect(editButtons[1]).toBeEnabled();
  });
});
