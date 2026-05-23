import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Stack,
  TextField,
} from '@mui/material';
import { FC, useEffect, useState } from 'react';
import { enqueueSnackbar } from 'notistack';
import { LogParserPreset, presetOperatorYaml } from 'api/logParserPresets';
import { useAddLogParserPreset, useChangeLogParserPreset } from 'hooks/api/useLogParserPresets';
import { apiErrorMessage } from 'utils/apiErrorMessage';
import { Messages } from '../../Settings.messages';

const PRESET_NAME_RE = /^[a-zA-Z][a-zA-Z0-9_]*$/;

export const DEFAULT_OPERATOR_YAML = `- type: regex_parser
  regex: '^(?P<message>.*)$'
  parse_from: body
`;

export type PresetDialogState =
  | { mode: 'create' }
  | { mode: 'edit'; preset: LogParserPreset };

export const LogParserPresetDialog: FC<{
  state: PresetDialogState | null;
  onClose: () => void;
}> = ({ state, onClose }) => {
  const isCreate = state?.mode === 'create';
  const preset = state?.mode === 'edit' ? state.preset : null;
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [operatorYaml, setOperatorYaml] = useState(DEFAULT_OPERATOR_YAML);

  const addPreset = useAddLogParserPreset();
  const changePreset = useChangeLogParserPreset();
  const m = Messages.otel.presets;

  useEffect(() => {
    if (!state) return;
    if (state.mode === 'create') {
      setName('');
      setDescription('');
      setOperatorYaml(DEFAULT_OPERATOR_YAML);
    } else {
      setName(state.preset.name);
      setDescription(state.preset.description ?? '');
      setOperatorYaml(presetOperatorYaml(state.preset) || DEFAULT_OPERATOR_YAML);
    }
  }, [state]);

  const save = async () => {
    try {
      if (isCreate) {
        if (!PRESET_NAME_RE.test(name.trim())) {
          enqueueSnackbar(m.invalidName, { variant: 'error' });
          return;
        }
        await addPreset.mutateAsync({
          name: name.trim(),
          description: description.trim(),
          operatorYaml: operatorYaml.trim(),
        });
      } else if (preset) {
        await changePreset.mutateAsync({
          id: preset.id,
          description: description.trim(),
          operatorYaml: operatorYaml.trim(),
        });
      }
      enqueueSnackbar(m.saved, { variant: 'success' });
      onClose();
    } catch (err) {
      enqueueSnackbar(apiErrorMessage(err, Messages.unauthorized), { variant: 'error' });
    }
  };

  const pending = addPreset.isPending || changePreset.isPending;

  return (
    <Dialog open={state !== null} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>{isCreate ? m.createTitle : m.editTitle}</DialogTitle>
      <DialogContent>
        <Stack gap={2} sx={{ mt: 1 }}>
          <TextField
            label={m.nameLabel}
            value={name}
            onChange={(e) => setName(e.target.value)}
            size="small"
            fullWidth
            disabled={!isCreate}
            helperText={isCreate ? m.nameHelp : undefined}
          />
          <TextField
            label={m.descriptionLabel}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            size="small"
            fullWidth
            multiline
            minRows={2}
          />
          <TextField
            label={m.yamlLabel}
            value={operatorYaml}
            onChange={(e) => setOperatorYaml(e.target.value)}
            size="small"
            fullWidth
            multiline
            minRows={10}
            slotProps={{
              input: {
                sx: {
                  fontFamily: 'Roboto Mono, monospace',
                  fontSize: '0.85rem',
                  whiteSpace: 'pre',
                  overflowX: 'auto',
                },
              },
            }}
            helperText={m.yamlHelp}
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{m.cancel}</Button>
        <Button variant="contained" onClick={save} disabled={pending}>
          {pending ? m.saving : m.save}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
