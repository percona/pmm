import {
  Box,
  Button,
  Chip,
  IconButton,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import AddIcon from '@mui/icons-material/Add';
import { FC, useMemo, useState } from 'react';
import { enqueueSnackbar } from 'notistack';
import { LogParserPreset, presetBuiltIn, presetUsageCount } from 'api/logParserPresets';
import {
  useLogParserPresets,
  useRemoveLogParserPreset,
} from 'hooks/api/useLogParserPresets';
import { apiErrorMessage } from 'utils/apiErrorMessage';
import { Messages } from '../../Settings.messages';
import { LogParserPresetDialog, PresetDialogState } from './LogParserPresetDialog';

export const LogParserPresetsSection: FC = () => {
  const { data: presets = [], isLoading, isError } = useLogParserPresets();
  const removePreset = useRemoveLogParserPreset();
  const [dialog, setDialog] = useState<PresetDialogState | null>(null);

  const counts = useMemo(() => {
    const builtIn = presets.filter((p) => presetBuiltIn(p)).length;
    return { builtIn, custom: presets.length - builtIn };
  }, [presets]);

  const m = Messages.otel.presets;

  const onDelete = async (preset: LogParserPreset) => {
    if (presetBuiltIn(preset)) return;
    if (!window.confirm(m.deleteConfirm(preset.name))) return;
    try {
      await removePreset.mutateAsync(preset.id);
      enqueueSnackbar(m.deleted, { variant: 'success' });
    } catch (err) {
      enqueueSnackbar(apiErrorMessage(err, Messages.unauthorized), { variant: 'error' });
    }
  };

  return (
    <Box>
      <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 2 }}>
        <Box>
          <Typography variant="h6">{m.title}</Typography>
          <Typography variant="body2" color="text.secondary">
            {m.summary(counts.builtIn, counts.custom)} {m.rawNote}
          </Typography>
        </Box>
        <Button startIcon={<AddIcon />} variant="outlined" onClick={() => setDialog({ mode: 'create' })}>
          {m.addButton}
        </Button>
      </Stack>

      {isLoading && (
        <Typography variant="body2" color="text.secondary">
          {m.loading}
        </Typography>
      )}
      {isError && (
        <Typography variant="body2" color="error">
          {m.loadError}
        </Typography>
      )}

      {!isLoading && !isError && (
        <Stack gap={1}>
          {presets.map((preset) => (
            <Stack
              key={preset.id}
              direction="row"
              alignItems="center"
              gap={1}
              sx={{ py: 1, px: 1.5, borderRadius: 1, border: 1, borderColor: 'divider' }}
            >
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <Stack direction="row" alignItems="center" gap={1} flexWrap="wrap">
                  <Typography variant="subtitle2">{preset.name}</Typography>
                  <Chip
                    size="small"
                    label={presetBuiltIn(preset) ? m.builtIn : m.custom}
                    variant="outlined"
                  />
                  {presetUsageCount(preset) > 0 && (
                    <Chip size="small" label={m.usedBy(presetUsageCount(preset))} variant="outlined" />
                  )}
                </Stack>
                {!!preset.description && (
                  <Typography variant="body2" color="text.secondary" noWrap title={preset.description}>
                    {preset.description}
                  </Typography>
                )}
              </Box>
              <Tooltip title={m.edit}>
                <IconButton size="small" onClick={() => setDialog({ mode: 'edit', preset })}>
                  <EditOutlinedIcon fontSize="small" />
                </IconButton>
              </Tooltip>
              {!presetBuiltIn(preset) && (
                <Tooltip title={presetUsageCount(preset) > 0 ? m.deleteBlocked : m.delete}>
                  <span>
                    <IconButton
                      size="small"
                      onClick={() => onDelete(preset)}
                      disabled={removePreset.isPending || presetUsageCount(preset) > 0}
                    >
                      <DeleteOutlineIcon fontSize="small" />
                    </IconButton>
                  </span>
                </Tooltip>
              )}
            </Stack>
          ))}
        </Stack>
      )}

      <LogParserPresetDialog state={dialog} onClose={() => setDialog(null)} />
    </Box>
  );
};
