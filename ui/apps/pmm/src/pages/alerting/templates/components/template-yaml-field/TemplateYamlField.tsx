import { ChangeEvent, FC, useRef } from 'react';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import UploadFileOutlinedIcon from '@mui/icons-material/UploadFileOutlined';
import { TextInput } from '@percona/percona-ui';
import { useFormContext } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import { readFileAsText } from '../../modal-create-template/CreateTemplateModal.utils';

interface Props {
  label: string;
  uploadLabel: string;
  placeholder?: string;
  disabled?: boolean;
}

export const TemplateYamlField: FC<Props> = ({
  label,
  uploadLabel,
  placeholder,
  disabled,
}) => {
  const { setValue } = useFormContext<{ yaml: string }>();
  const inputRef = useRef<HTMLInputElement>(null);

  const handleFile = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) {
      return;
    }
    try {
      const text = await readFileAsText(file);
      setValue('yaml', text, { shouldDirty: true, shouldValidate: true });
    } catch {
      enqueueSnackbar('Failed to read file', { variant: 'error' });
    }
  };

  return (
    <Stack gap={1}>
      <Stack direction="row" justifyContent="flex-end">
        <Button
          type="button"
          size="small"
          variant="text"
          startIcon={<UploadFileOutlinedIcon />}
          disabled={disabled}
          data-testid="upload-template-yaml"
          onClick={() => inputRef.current?.click()}
        >
          {uploadLabel}
        </Button>
        <input
          ref={inputRef}
          type="file"
          accept=".yml,.yaml"
          hidden
          data-testid="template-yaml-file-input"
          onChange={handleFile}
        />
      </Stack>
      <TextInput
        name="yaml"
        label={label}
        textFieldProps={{
          multiline: true,
          minRows: 10,
          placeholder,
          disabled,
          slotProps: { htmlInput: { 'data-testid': 'template-yaml' } },
        }}
      />
    </Stack>
  );
};

export default TemplateYamlField;
