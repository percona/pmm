import { FC } from 'react';
import Divider from '@mui/material/Divider';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { SelectInput, TextInput } from '@percona/percona-ui';
import { useFormContext } from 'react-hook-form';
import { DashboardFolder } from 'types/folders.types';
import { CreateRuleFormValues } from '../CreateAlertFromTemplate.types';
import { CREATE_FOLDER_VALUE } from '../CreateAlertFromTemplate.constants';
import { Messages } from '../CreateAlertFromTemplate.messages';

interface Props {
  folders: DashboardFolder[];
  loadingFolders?: boolean;
}

export const FolderGroupSection: FC<Props> = ({ folders, loadingFolders }) => {
  const { watch } = useFormContext<CreateRuleFormValues>();
  const creatingFolder = watch('folderUid') === CREATE_FOLDER_VALUE;

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
      <TextInput name="group" label={Messages.fields.group} isRequired />
      <TextInput
        name="interval"
        label={Messages.fields.interval}
        isRequired
        textFieldProps={{ type: 'number' }}
      />
    </Stack>
  );
};

export default FolderGroupSection;
