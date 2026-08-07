import { FC, useEffect, useMemo, useRef, useState } from 'react';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { SelectInput } from '@percona/percona-ui';
import { useFormContext } from 'react-hook-form';
import { DashboardFolder } from 'types/folders.types';
import { RulerRuleGroup } from 'types/alert-rule-groups.types';
import { useFolderRuleGroups } from 'hooks/api/useFolderRuleGroups';
import { CreateRuleFormValues } from '../CreateAlertFromTemplate.types';
import { DEFAULT_INTERVAL } from '../CreateAlertFromTemplate.constants';
import { Messages } from '../CreateAlertFromTemplate.messages';
import { NewEvaluationGroupModal } from './NewEvaluationGroupModal';
import { NewFolderModal } from './NewFolderModal';

interface Props {
  folders: DashboardFolder[];
  loadingFolders?: boolean;
}

export const FolderGroupSection: FC<Props> = ({ folders, loadingFolders }) => {
  const { watch, setValue } = useFormContext<CreateRuleFormValues>();
  const folderUid = watch('folderUid');
  const group = watch('group');
  const interval = watch('interval');
  const canSetGroup = !!folderUid;

  const { data: fetchedGroups = [], isLoading: loadingGroups } =
    useFolderRuleGroups(folderUid, { enabled: canSetGroup });

  // Folders / groups created via their modals this session (merged with the
  // fetched lists so the just-created value is immediately selectable).
  const [createdFolders, setCreatedFolders] = useState<DashboardFolder[]>([]);
  const [createdGroups, setCreatedGroups] = useState<RulerRuleGroup[]>([]);
  const [folderModalOpen, setFolderModalOpen] = useState(false);
  const [groupModalOpen, setGroupModalOpen] = useState(false);

  const allFolders = useMemo(() => {
    const byUid = new Map<string, DashboardFolder>();
    [...folders, ...createdFolders].forEach((f) => byUid.set(f.uid, f));
    return Array.from(byUid.values());
  }, [folders, createdFolders]);

  // Reset the group whenever the folder changes (not on first mount).
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

  const handleCreateFolder = (folder: DashboardFolder) => {
    setCreatedFolders((prev) => [
      ...prev.filter((f) => f.uid !== folder.uid),
      folder,
    ]);
    setValue('folderUid', folder.uid, { shouldValidate: true });
  };

  const handleCreateGroup = (name: string, groupInterval: string) => {
    setCreatedGroups((prev) => [
      ...prev.filter((g) => g.name !== name),
      { name, interval: groupInterval },
    ]);
    setValue('group', name, { shouldValidate: true });
    setValue('interval', groupInterval, { shouldValidate: true });
    setGroupModalOpen(false);
  };

  return (
    <Stack gap={2}>
      <Typography variant="h6">{Messages.sections.location}</Typography>

      <Stack direction="row" gap={1} alignItems="center">
        <Box sx={{ flex: 1 }}>
          <SelectInput
            name="folderUid"
            label={Messages.fields.folder}
            isRequired
            loading={loadingFolders}
            formControlProps={{ fullWidth: true, sx: { mt: 0 } }}
          >
            {allFolders.map((folder) => (
              <MenuItem key={folder.uid} value={folder.uid}>
                {folder.title}
              </MenuItem>
            ))}
          </SelectInput>
        </Box>
        <Typography variant="body2" color="text.secondary">
          {Messages.group.or}
        </Typography>
        <Button
          type="button"
          variant="text"
          startIcon={<AddOutlinedIcon />}
          data-testid="new-folder"
          onClick={() => setFolderModalOpen(true)}
        >
          {Messages.fields.newFolder}
        </Button>
      </Stack>

      <Stack direction="row" gap={1} alignItems="center">
        <Box sx={{ flex: 1 }}>
          <SelectInput
            name="group"
            label={
              canSetGroup ? Messages.group.label : Messages.group.labelNoFolder
            }
            isRequired
            loading={loadingGroups}
            formControlProps={{ fullWidth: true, sx: { mt: 0 } }}
            selectFieldProps={{ disabled: !canSetGroup }}
          >
            {groups.map((g) => (
              <MenuItem key={g.name} value={g.name}>
                {g.name} ({g.interval ?? DEFAULT_INTERVAL})
              </MenuItem>
            ))}
          </SelectInput>
        </Box>
        <Typography variant="body2" color="text.secondary">
          {Messages.group.or}
        </Typography>
        <Button
          type="button"
          variant="text"
          startIcon={<AddOutlinedIcon />}
          disabled={!canSetGroup}
          data-testid="new-eval-group"
          onClick={() => setGroupModalOpen(true)}
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

      <NewFolderModal
        open={folderModalOpen}
        onClose={() => setFolderModalOpen(false)}
        onCreated={handleCreateFolder}
      />
      <NewEvaluationGroupModal
        open={groupModalOpen}
        onClose={() => setGroupModalOpen(false)}
        onCreate={handleCreateGroup}
      />
    </Stack>
  );
};

export default FolderGroupSection;
