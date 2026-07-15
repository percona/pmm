import { FC } from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { TestWrapper } from 'utils/testWrapper';
import { Severity } from 'types/alert-templates.types';
import { DashboardFolder } from 'types/folders.types';
import * as groupsHook from 'hooks/api/useFolderRuleGroups';
import { CreateRuleFormValues } from '../CreateAlertFromTemplate.types';
import { FolderGroupSection } from './FolderGroupSection';

vi.mock('hooks/api/useFolderRuleGroups');

const useFolderRuleGroupsMock = vi.mocked(groupsHook.useFolderRuleGroups);

const FOLDERS: DashboardFolder[] = [
  { id: 1, uid: 'folder-1', title: 'MySQL alerts' },
];

const Harness: FC<{ folderUid?: string; group?: string }> = ({
  folderUid = 'folder-1',
  group = '',
}) => {
  const methods = useForm<CreateRuleFormValues>({
    defaultValues: {
      template: 'tpl',
      name: 'rule',
      severity: Severity.WARNING,
      duration: '60',
      folderUid,
      newFolderTitle: '',
      group,
      interval: '1m',
      filters: [],
      params: {},
    },
  });
  return (
    <FormProvider {...methods}>
      <FolderGroupSection folders={FOLDERS} />
    </FormProvider>
  );
};

const renderSection = (folderUid?: string, group?: string) =>
  render(
    <TestWrapper>
      <Harness folderUid={folderUid} group={group} />
    </TestWrapper>
  );

describe('FolderGroupSection', () => {
  beforeEach(() => {
    useFolderRuleGroupsMock.mockReturnValue({
      data: [
        { name: 'group-a', interval: '5m' },
        { name: 'group-b', interval: '30s' },
      ],
      isLoading: false,
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);
  });

  it('disables the group control until a folder is chosen', () => {
    renderSection('');
    expect(
      screen.getAllByText(
        'Select a folder before setting evaluation group and interval'
      ).length
    ).toBeGreaterThan(0);
    expect(screen.getByTestId('new-eval-group')).toBeDisabled();
  });

  it('locks the interval to the selected existing group', async () => {
    // Pre-select an existing group; the sync effect should lock the interval.
    renderSection('folder-1', 'group-a');

    await waitFor(() =>
      expect(screen.getByTestId('eval-interval-text')).toHaveTextContent(
        'All rules in the selected group are evaluated every 5m.'
      )
    );
  });

  it('creates a new evaluation group via the modal', async () => {
    renderSection('folder-1');
    fireEvent.click(screen.getByTestId('new-eval-group'));

    fireEvent.change(await screen.findByTestId('new-eval-group-name'), {
      target: { value: 'my-new-group' },
    });
    fireEvent.click(screen.getByTestId('interval-preset-10m'));

    const createButton = screen.getByTestId('new-eval-group-create');
    await waitFor(() => expect(createButton).toBeEnabled());
    fireEvent.click(createButton);

    await waitFor(() =>
      expect(screen.getByTestId('eval-interval-text')).toHaveTextContent(
        'All rules in the selected group are evaluated every 10m.'
      )
    );
  });
});
