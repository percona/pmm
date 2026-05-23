import {
  Button,
  FormControlLabel,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material';
import { FC, useState } from 'react';
import { ADRE_BEHAVIOR_CONTROL_KEYS } from 'api/adre';
import {
  AdreBehaviorVariant,
  effectiveValue,
  labelForKey,
  shippedPreset,
} from './AdreBehaviorControlsBlock.utils';

export interface AdreBehaviorControlsBlockProps {
  variant: AdreBehaviorVariant;
  title: string;
  description: string;
  value: Record<string, boolean>;
  onChange: (next: Record<string, boolean>) => void;
  onJsonError: (message: string) => void;
}

export const AdreBehaviorControlsBlock: FC<AdreBehaviorControlsBlockProps> = ({
  variant,
  title,
  description,
  value,
  onChange,
  onJsonError,
}) => {
  const [jsonDraft, setJsonDraft] = useState<string | null>(null);
  const jsonShown = jsonDraft ?? JSON.stringify(value, null, 2);

  const setKey = (key: string, checked: boolean) => {
    onChange({ ...value, [key]: checked });
  };

  return (
    <Stack gap={1.5}>
      <Typography variant="subtitle2" fontWeight={600}>
        {title}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        {description}{' '}
        See your AI service documentation for{' '}
        fast mode / prompt controls
        . Clearing the map to <code>{'{}'}</code> in Advanced JSON makes PMM use the shipped preset for that mode when calling the AI backend. On the AI service container,{' '}
        <code>ENABLED_PROMPTS</code> can still override what the API enables.
      </Typography>
      <Stack gap={0.5} sx={{ pl: 0.5 }}>
        {ADRE_BEHAVIOR_CONTROL_KEYS.map((key) => (
          <FormControlLabel
            key={key}
            control={
              <Switch
                size="small"
                checked={effectiveValue(value, key, variant)}
                onChange={(_e, checked) => setKey(key, checked)}
              />
            }
            label={labelForKey(key)}
          />
        ))}
      </Stack>
      <TextField
        label="Advanced JSON"
        value={jsonShown}
        onChange={(e) => setJsonDraft(e.target.value)}
        onBlur={() => {
          if (jsonDraft == null) return;
          try {
            const parsed = JSON.parse(jsonDraft) as unknown;
            if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
              throw new Error('Root value must be a JSON object');
            }
            const next: Record<string, boolean> = {};
            for (const [k, v] of Object.entries(parsed)) {
              if (typeof v !== 'boolean') {
                throw new Error(`Key "${k}" must be a boolean`);
              }
              next[k] = v;
            }
            onChange(next);
            setJsonDraft(null);
          } catch (e) {
            onJsonError(e instanceof Error ? e.message : 'Invalid JSON');
            setJsonDraft(null);
          }
        }}
        size="small"
        fullWidth
        multiline
        minRows={4}
        sx={{ fontFamily: 'monospace' }}
      />
      <Button
        size="small"
        variant="outlined"
        onClick={() => {
          onChange(shippedPreset(variant));
          setJsonDraft(null);
        }}
      >
        Reset to preset
      </Button>
    </Stack>
  );
};
