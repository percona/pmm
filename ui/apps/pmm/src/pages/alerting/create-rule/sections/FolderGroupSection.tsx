import { FC, useEffect, useMemo, useRef, useState } from 'react';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { SelectInput, TextInput } from '@percona/percona-ui';
import { useFormContext } from 'react-hook-form';
import { DashboardFolder } from 'types/folders.types';
import { RulerRuleGroup } from 'types/alert-rule-groups.types';
import { useFolderRuleGroups } from 'hooks/api/useFolderRuleGroups';
import { CreateRuleFormValues } from '../CreateAlertFromTemplate.types';
import {
  CREATE_FOLDER_VALUE,
  DEFAULT_INTERVAL,
} from '../CreateAlertFromTemplate.constants';
import { Messages } from '../CreateAlertFromTemplate.messages';
import { NewEvaluationGroupModal } from './NewEvaluationGroupModal';

interface Props {
  folders: DashboardFolder[];
  loadingFolders?: boolean;
}

export const FolderGroupSection: FC<Props> = ({ folders, loadingFolders }) => {
  const { watch, setValue } = useFormContext<CreateRuleFormValues>();
  const folderUid = watch('folderUid');
  const group = watch('group');
  const interval = watch('interval');

  const creatingFolder = folderUid === CREATE_FOLDER_VALUE;
  const existingFolderSelected = !!folderUid && !creatingFolder;
  // The evaluation group can be set once a target folder is known (an existing
  // folder, or a new folder being created on submit).
  const canSetGroup = existingFolderSelected || creatingFolder;

  const { data: fetchedGroups = [], isLoading: loadingGroups } =
    useFolderRuleGroups(folderUid, { enabled: existingFolderSelected });

  // Groups created via the modal this session (not yet persisted server-side).
  const [createdGroups, setCreatedGroups] = useState<RulerRuleGroup[]>([]);
  const [modalOpen, setModalOpen] = useState(false);

  // Reset the group selection whenever the target folder changes (but not on
  // first mount, so a seeded selection survives the initial render).
  const previousFolder = useRef(folderUid);
  useEffect(() => {
    if (previousFolder.current === folderUid) {
      return;
    }
    previousFolder.current = folderUid;
    setCreatedGroups([]);
    setValue('group', '');
    setValue('interval', DEFAULT_INTERVAL);
  }, [folderUid, setValue]);

  const groups = useMemo(() => {
    const byName = new Map<string, RulerRuleGroup>();
    [...fetchedGroups, ...createdGroups].forEach((g) => byName.set(g.name, g));
    return Array.from(byName.values()).sort((a, b) =>
      a.name.localeCompare(b.name)
    );
  }, [fetchedGroups, createdGroups]);

  // Mirror Grafana: selecting an existing group locks the interval to it.
  useEffect(() => {
    const selected = groups.find((g) => g.name === group);
    if (selected) {
      setValue('interval', selected.interval ?? DEFAULT_INTERVAL, {
        shouldValidate: true,
      });
    }
  }, [group, groups, setValue]);

  const handleCreateGroup = (name: string, groupInterval: string) => {
    setCreatedGroups((prev) => [
      ...prev.filter((g) => g.name !== name),
      { name, interval: groupInterval },
    ]);
    setValue('group', name, { shouldValidate: true });
    setValue('interval', groupInterval, { shouldValidate: true });
    setModalOpen(false);
  };

  return (
    <Stack gap={2}>
      <Typography variant="h6">{Messages.sections.location}</Typography>
      <SelectInput
        name="folderUid"
        label={Messages.fields.folder}
        isRequired
        loading={loadingFolders}
      >
        {folders.map((folder) => (
          <MenuItem key={folder.uid} value={folder.uid}>
            {folder.title}
          </MenuItem>
        ))}
        <Divider />
        <MenuItem value={CREATE_FOLDER_VALUE}>
          {Messages.fields.newFolder}
        </MenuItem>
      </SelectInput>
      {creatingFolder && (
        <TextInput
          name="newFolderTitle"
          label={Messages.fields.newFolderTitle}
          isRequired
        />
      )}

      <SelectInput
        name="group"
        label={
          canSetGroup ? Messages.group.label : Messages.group.labelNoFolder
        }
        isRequired
        loading={loadingGroups}
        selectFieldProps={{
          displayEmpty: true,
          disabled: !canSetGroup,
          renderValue: (value) =>
            (value as string) || Messages.group.placeholder,
        }}
      >
        {groups.map((g) => (
          <MenuItem key={g.name} value={g.name}>
            {g.name} ({g.interval ?? DEFAULT_INTERVAL})
          </MenuItem>
        ))}
      </SelectInput>
      <Stack direction="row" alignItems="center" gap={1}>
        <Typography variant="body2" color="text.secondary">
          {Messages.group.or}
        </Typography>
        <Button
          type="button"
          variant="text"
          size="small"
          startIcon={<AddOutlinedIcon />}
          disabled={!canSetGroup}
          data-testid="new-eval-group"
          onClick={() => setModalOpen(true)}
        >
          {Messages.group.newGroup}
        </Button>
      </Stack>
      {!!group && (
        <Typography
          variant="body2"
          color="text.secondary"
          data-testid="eval-interval-text"
        >
          {Messages.group.evaluatedEvery(interval)}
        </Typography>
      )}

      <NewEvaluationGroupModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onCreate={handleCreateGroup}
      />
    </Stack>
  );
};

export default FolderGroupSection;
