import { FC } from 'react';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { SelectInput, TextInput } from '@percona/percona-ui';
import { DashboardFolder } from 'types/folders.types';
import { Messages } from '../CreateAlertFromTemplate.messages';

interface Props {
  folders: DashboardFolder[];
  loadingFolders?: boolean;
}

export const FolderGroupSection: FC<Props> = ({ folders, loadingFolders }) => (
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
    </SelectInput>
    <TextInput name="group" label={Messages.fields.group} isRequired />
    <TextInput
      name="interval"
      label={Messages.fields.interval}
      isRequired
      textFieldProps={{ type: 'number' }}
    />
  </Stack>
);

export default FolderGroupSection;
