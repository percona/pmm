/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { useRef, useState, type ReactNode } from 'react';
import {
  Box,
  ButtonBase,
  IconButton,
  Stack,
  TextField,
  Tooltip,
  Typography,
  type TypographyProps,
} from '@mui/material';
import CheckIcon from '@mui/icons-material/Check';
import CloseIcon from '@mui/icons-material/Close';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';

export interface InlineEditableTextProps {
  value: string;
  /** Accessible name of the field, e.g. "Incident name". */
  label: string;
  /** Called with the trimmed value; only when it differs from `value`. */
  onSave: (value: string) => void;
  /** An empty value is refused rather than saved. */
  required?: boolean;
  /** What reads in place of an empty value, as the prompt to fill it. */
  placeholder?: ReactNode;
  disabled?: boolean;
  saving?: boolean;
  /** Controlled editing, so a caller (an actions menu) can start it. */
  editing?: boolean;
  onEditingChange?: (editing: boolean) => void;
  variant?: TypographyProps['variant'];
  /** Rendered as the display element, so a page title stays its heading. */
  component?: React.ElementType;
  testId?: string;
}

/**
 * Text that reads as text and becomes a field when clicked. Enter or the check
 * saves, Escape or the cross cancels, and leaving the field saves — a title
 * someone typed and then clicked away from was meant, not abandoned.
 */
export function InlineEditableText({
  value,
  label,
  onSave,
  required = false,
  placeholder,
  disabled = false,
  saving = false,
  editing: editingProp,
  onEditingChange,
  variant = 'body1',
  component,
  testId,
}: InlineEditableTextProps) {
  const [editingState, setEditingState] = useState(false);
  const editing = editingProp ?? editingState;
  const [draft, setDraft] = useState(value);
  // Seeded on the way into editing, whoever started it: a caller flipping
  // `editing` from outside never goes through `setEditing`.
  const [wasEditing, setWasEditing] = useState(editing);
  if (editing !== wasEditing) {
    setWasEditing(editing);
    if (editing) {
      setDraft(value);
    }
  }
  // Enter and the check both end the edit, and the field's unmount can still
  // deliver a blur afterwards; one edit saves once.
  const settledRef = useRef(false);

  const setEditing = (next: boolean) => {
    settledRef.current = !next;
    setEditingState(next);
    onEditingChange?.(next);
  };

  const commit = () => {
    if (settledRef.current) {
      return;
    }
    const next = draft.trim();
    if (!(required && next === '') && next !== value) {
      onSave(next);
    }
    setEditing(false);
  };

  const cancel = () => {
    if (settledRef.current) {
      return;
    }
    setEditing(false);
  };

  if (editing && !disabled) {
    return (
      <Stack direction="row" alignItems="center" spacing={0.5}>
        <TextField
          autoFocus
          size="small"
          value={draft}
          disabled={saving}
          onFocus={(event) => {
            // A fresh edit, however it started (here or from a caller).
            settledRef.current = false;
            event.target.select();
          }}
          onChange={(event) => setDraft(event.target.value)}
          onBlur={commit}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              event.preventDefault();
              commit();
            } else if (event.key === 'Escape') {
              event.preventDefault();
              cancel();
            }
          }}
          slotProps={{
            htmlInput: {
              'aria-label': label,
              'data-testid': testId ? `${testId}-input` : undefined,
            },
          }}
          sx={{ minWidth: 240 }}
        />
        <Tooltip title="Save">
          <IconButton
            size="small"
            aria-label={`Save ${label.toLowerCase()}`}
            // Before the field's blur, so the click is not lost to a re-render.
            onMouseDown={(event) => event.preventDefault()}
            onClick={commit}
            disabled={saving || (required && draft.trim() === '')}
          >
            <CheckIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <Tooltip title="Cancel">
          <IconButton
            size="small"
            aria-label={`Cancel editing ${label.toLowerCase()}`}
            onMouseDown={(event) => event.preventDefault()}
            onClick={cancel}
          >
            <CloseIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      </Stack>
    );
  }

  const content = (
    <>
      {value || (
        <Box component="span" sx={{ color: 'text.secondary' }}>
          {placeholder}
        </Box>
      )}
    </>
  );

  if (disabled) {
    return (
      <Typography
        variant={variant}
        component={component ?? 'span'}
        data-testid={testId}
      >
        {content}
      </Typography>
    );
  }

  // The heading wraps the button, not the other way round: a button may only
  // hold phrasing content, and the page title has to stay its heading.
  return (
    <Typography
      variant={variant}
      component={component ?? 'span'}
      data-testid={testId}
    >
      {/*
        Described, not labelled: a label would replace the value as the
        button's name, and with it the heading's.
      */}
      <Tooltip title={`Edit ${label.toLowerCase()}`} describeChild>
        <ButtonBase
          onClick={() => setEditing(true)}
          sx={{
            font: 'inherit',
            gap: 1,
            px: 0.5,
            mx: -0.5,
            borderRadius: 1,
            textAlign: 'left',
            '& .atw-inline-edit-icon': { opacity: 0 },
            '&:hover, &.Mui-focusVisible': { bgcolor: 'action.hover' },
            '&:hover .atw-inline-edit-icon, &.Mui-focusVisible .atw-inline-edit-icon':
              { opacity: 1 },
          }}
        >
          {content}
          <Box
            component="span"
            className="atw-inline-edit-icon"
            sx={{ display: 'inline-flex', color: 'text.secondary' }}
          >
            <EditOutlinedIcon fontSize="small" />
          </Box>
        </ButtonBase>
      </Tooltip>
    </Typography>
  );
}
