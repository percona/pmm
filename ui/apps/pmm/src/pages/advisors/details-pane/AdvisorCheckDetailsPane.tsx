import { FC, ReactNode, useEffect, useState } from 'react';
import CloseIcon from '@mui/icons-material/Close';
import CloseFullscreenOutlinedIcon from '@mui/icons-material/CloseFullscreenOutlined';
import ContentCopyOutlinedIcon from '@mui/icons-material/ContentCopyOutlined';
import ControlPointDuplicateOutlinedIcon from '@mui/icons-material/ControlPointDuplicateOutlined';
import OpenInFullOutlinedIcon from '@mui/icons-material/OpenInFullOutlined';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Divider from '@mui/material/Divider';
import Drawer from '@mui/material/Drawer';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import { CopyToClipboardButton } from '@percona/percona-ui';
import { enqueueSnackbar } from 'notistack';
import {
  DRAWER_CLOSED_WIDTH,
  DRAWER_WIDTH,
} from 'components/sidebar/drawer/Drawer.constants';
import { useNavigation } from 'contexts/navigation/navigation.hooks';
import { useAdvisorCheckScript } from 'hooks/api/useAdvisors';
import { AdvisorCheckRow } from 'types/advisors.types';
import { ADVISOR_FAMILY, ADVISOR_INTERVAL } from 'lib/constants';
import { ScriptEditorInput } from '../check-form/ScriptEditorInput';
import { Messages } from '../AdvisorsList.messages';

const EM_DASH = '—';

interface FieldProps {
  label: string;
  children: ReactNode;
  // grid columns to span (out of 4)
  span?: number;
}

const Field: FC<FieldProps> = ({ label, children, span = 1 }) => (
  <Stack
    gap={0.5}
    sx={{ gridColumn: { xs: 'span 4', md: `span ${span}` } }}
    data-testid={`check-details-field-${label.toLowerCase().replace(/[^a-z]+/g, '-')}`}
  >
    <Typography variant="caption" color="text.secondary">
      {label}
    </Typography>
    <Box
      sx={{
        flex: 1,
        display: 'flex',
        alignItems: 'flex-end',
        // percona-ui's theme pins MuiIconButton to a fixed 40x40 (BaseTheme
        // sizeMedium), which stretches the row and lifts the value off the
        // underline. Shrink the copy button to its icon so it matches the
        // plain fields. (Full `padding` property, not the `p` shorthand: MUI
        // sx does not expand shorthands inside nested selector keys.)
        '& .MuiIconButton-root': { width: 'auto', height: 'auto', padding: 0 },
      }}
    >
      {children}
    </Box>
    <Divider />
  </Stack>
);

interface AdvisorCheckDetailsPaneProps {
  check: AdvisorCheckRow | null;
  // open the pane already maximized (e.g. triggered by a row double-click)
  initialMaximized?: boolean;
  onClose: () => void;
  // clone the current check into a new editable check
  onClone?: () => void;
}

export const AdvisorCheckDetailsPane: FC<AdvisorCheckDetailsPaneProps> = ({
  check,
  initialMaximized = false,
  onClose,
  onClone,
}) => {
  const [maximized, setMaximized] = useState(false);
  const { navOpen } = useNavigation();
  const open = !!check;
  // the pane never covers the main navigation
  const sidebarWidth = navOpen ? DRAWER_WIDTH : DRAWER_CLOSED_WIDTH;

  const {
    data: script,
    isLoading: isScriptLoading,
    isError: isScriptError,
  } = useAdvisorCheckScript(check?.checkName);

  // apply the requested height on each open, reset on close
  useEffect(() => {
    setMaximized(open ? initialMaximized : false);
  }, [open, initialMaximized]);

  const m = Messages.details;

  const handleCopyCode = () => {
    if (!script) {
      return;
    }
    void navigator.clipboard.writeText(script);
    enqueueSnackbar(m.codeCopied, { variant: 'success' });
  };

  return (
    <Drawer
      anchor="bottom"
      open={open}
      onClose={onClose}
      slotProps={{
        paper: {
          // @ts-expect-error data-testid is passed through to the DOM
          'data-testid': 'check-details-pane',
          sx: {
            height: maximized ? '100vh' : '60vh',
            left: { xs: 0, md: sidebarWidth },
            p: 2,
            // column layout so the code area can flex-fill and scroll internally
            display: 'flex',
            flexDirection: 'column',
            transition: (theme) =>
              theme.transitions.create(['height', 'left'], {
                duration: theme.transitions.duration.short,
              }),
          },
        },
      }}
    >
      <Stack direction="row" justifyContent="space-between" alignItems="center">
        <Typography variant="h6">{m.title}</Typography>
        <Stack direction="row" gap={1}>
          <IconButton
            size="small"
            aria-label={m.maximize}
            onClick={() => setMaximized((current) => !current)}
            data-testid="check-details-maximize"
          >
            {maximized ? (
              <CloseFullscreenOutlinedIcon fontSize="small" />
            ) : (
              <OpenInFullOutlinedIcon fontSize="small" />
            )}
          </IconButton>
          <IconButton
            size="small"
            aria-label={m.close}
            onClick={onClose}
            data-testid="check-details-close"
          >
            <CloseIcon fontSize="small" />
          </IconButton>
        </Stack>
      </Stack>
      {check && (
        <Box
          sx={{
            flex: 1,
            minHeight: 0,
            mt: 2,
            display: 'flex',
            flexDirection: 'column',
            gap: 3,
            overflow: 'auto',
          }}
        >
          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: 'repeat(4, 1fr)',
              columnGap: 4,
              rowGap: 3,
            }}
          >
            <Field label={m.checkName}>
              <Stack direction="row" alignItems="center" gap={0.5}>
                <Typography variant="body1">{check.checkName}</Typography>
                <CopyToClipboardButton textToCopy={check.checkName} />
              </Stack>
            </Field>
            <Field label={m.category}>
              <Typography variant="body1">{check.category}</Typography>
            </Field>
            <Field label={m.subcategory}>
              <Typography variant="body1">{check.subcategory}</Typography>
            </Field>
            <Field label={m.vendor}>
              <Typography variant="body1">
                {ADVISOR_FAMILY[check.family]}
              </Typography>
            </Field>
            <Field label={m.interval}>
              <Typography variant="body1">
                {ADVISOR_INTERVAL[check.interval]}
              </Typography>
            </Field>
            <Field label={m.status}>
              <Typography variant="body1">
                {check.enabled
                  ? Messages.status.enabled
                  : Messages.status.disabled}
              </Typography>
            </Field>

            <Field label={m.summary} span={4}>
              <Typography variant="body1">
                {check.summary || EM_DASH}
              </Typography>
            </Field>
            <Field label={m.description} span={4}>
              <Typography variant="body1">
                {check.description || EM_DASH}
              </Typography>
            </Field>
          </Box>

          <Stack gap={1} sx={{ flex: 1, minHeight: 0 }}>
            <Stack direction="row" justifyContent="flex-end" gap={1}>
              {onClone && (
                <Button
                  size="small"
                  startIcon={
                    <ControlPointDuplicateOutlinedIcon fontSize="small" />
                  }
                  onClick={onClone}
                  data-testid="check-clone"
                >
                  {m.clone}
                </Button>
              )}
              <Button
                size="small"
                startIcon={<ContentCopyOutlinedIcon fontSize="small" />}
                onClick={handleCopyCode}
                disabled={!script}
                data-testid="check-code-copy"
              >
                {m.copyCode}
              </Button>
            </Stack>
            {isScriptLoading ? (
              <CircularProgress size={24} data-testid="check-code-loading" />
            ) : isScriptError ? (
              <Typography variant="body2" color="error">
                {m.codeError}
              </Typography>
            ) : script ? (
              // same syntax-highlighted editor as the check form, read-only;
              // fills the remaining height and scrolls internally, so the pane
              // itself never scrolls
              <TextField
                // explicit id: MUI's auto-generated useId (":r1:") is not a
                // valid CSS identifier and breaks jsdom's selector matching
                id="check-details-script"
                label={m.script}
                value={script}
                multiline
                sx={{
                  flex: 1,
                  minHeight: 0,
                  '& .MuiInputBase-root': {
                    height: '100%',
                    alignItems: 'flex-start',
                    overflow: 'auto',
                  },
                  '& textarea': { outline: 'none' },
                }}
                slotProps={{
                  input: { inputComponent: ScriptEditorInput, readOnly: true },
                  htmlInput: { 'data-testid': 'check-code' },
                }}
              />
            ) : (
              <Typography variant="body2" color="text.secondary">
                {m.noCode}
              </Typography>
            )}
          </Stack>
        </Box>
      )}
    </Drawer>
  );
};
