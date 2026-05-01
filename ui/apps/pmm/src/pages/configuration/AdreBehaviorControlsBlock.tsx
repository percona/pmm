import {
  Button,
  FormControlLabel,
  Link,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material';
import { FC, useState } from 'react';
import { ADRE_BEHAVIOR_CONTROL_KEYS } from 'api/adre';

const HOLMES_PROMPT_CONTROLS =
  'https://holmesgpt.dev/dev/reference/http-api/?h=fast#fast-mode--prompt-controls';

const FAST_SHIPPED: Record<string, boolean> = {
  time_skills: false,
  todowrite_instructions: false,
  todowrite_reminder: false,
};

export type AdreBehaviorVariant = 'fast' | 'investigation' | 'format';

function shippedPreset(variant: AdreBehaviorVariant): Record<string, boolean> {
  if (variant === 'investigation') return {};
  return { ...FAST_SHIPPED };
}

/** Merge stored settings with PMM shipped preset for empty keys (editing model). */
export function hydrateAdreBehaviorMap(
  raw: Record<string, boolean> | undefined | null,
  variant: AdreBehaviorVariant
): Record<string, boolean> {
  return { ...shippedPreset(variant), ...(raw ?? {}) };
}

function effectiveValue(
  map: Record<string, boolean>,
  key: string,
  variant: AdreBehaviorVariant
): boolean {
  if (Object.prototype.hasOwnProperty.call(map, key)) return map[key];
  if (variant === 'investigation') return true;
  return shippedPreset(variant)[key] ?? true;
}

function labelForKey(key: string): string {
  return key
    .split('_')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

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
        <Link href={HOLMES_PROMPT_CONTROLS} target="_blank" rel="noreferrer">
          Holmes fast mode / prompt controls
        </Link>
        . Clearing the map to <code>{'{}'}</code> in Advanced JSON makes PMM use the shipped preset for that mode when calling Holmes. On the Holmes container,{' '}
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
