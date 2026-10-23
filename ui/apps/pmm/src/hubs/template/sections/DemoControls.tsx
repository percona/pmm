import TuneOutlined from '@mui/icons-material/TuneOutlined';
import Box from '@mui/material/Box';
import ClickAwayListener from '@mui/material/ClickAwayListener';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Popper from '@mui/material/Popper';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import Typography from '@mui/material/Typography';
import { Kbd, Tooltip } from '@percona/peak-ui';
import {
  type FC,
  type KeyboardEvent as ReactKeyboardEvent,
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
} from 'react';
import { ThemeScope } from 'components/theme-scope';
import { Messages } from '../hub.messages';
import { DEFAULT_HUB_PARAMS } from '../hub.params';
import type { HubParams } from '../hub.types';

const SCENARIOS = ['healthy', 'incomplete', 'unhealthy'] as const;
const FRESHNESS = ['live', 'stale'] as const;
const OPERATIONS = ['on', 'off'] as const;

/** Where a typed D belongs to the field, not to the shortcut. */
const isTyping = (target: EventTarget | null) =>
  target instanceof HTMLElement &&
  (target.isContentEditable ||
    ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName));

export interface DemoControlsProps {
  /** The technology's name, as in "Operations for MongoDB". */
  name: string;
  params: HubParams;
  setParams: (next: Partial<HubParams>) => void;
  /** Whether the page shows Operations, after the switches. */
  operations: boolean;
  /** Only an admin can have Operations on. */
  admin: boolean;
}

/** Prototype only: a floating button and a non-modal panel that switch the hub's demo states, all kept in the URL. */
export const DemoControls: FC<DemoControlsProps> = ({
  name,
  params,
  setParams,
  operations,
  admin,
}) => {
  const messages = Messages.demo;
  const panelId = useId();
  const titleId = useId();
  const groupId = useId();
  const [button, setButton] = useState<HTMLButtonElement | null>(null);
  const panelRef = useRef<HTMLDivElement | null>(null);
  const [open, setOpen] = useState(false);
  // The panel takes focus as it mounts, so the keyboard lands in it however it was opened.
  const attachPanel = useCallback((node: HTMLDivElement | null) => {
    panelRef.current = node;
    node?.focus();
  }, []);

  const close = (returnFocus: boolean) => {
    setOpen(false);
    if (returnFocus) {
      button?.focus();
    }
  };

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      // The physical key, so Caps Lock and the keyboard layout don't change it.
      if (
        event.code !== 'KeyD' ||
        !event.shiftKey ||
        event.metaKey ||
        event.ctrlKey ||
        event.altKey ||
        event.repeat ||
        isTyping(event.target)
      ) {
        return;
      }
      event.preventDefault();
      // The panel is mounted only while open.
      const panel = panelRef.current;

      if (!panel) {
        setOpen(true);

        return;
      }
      setOpen(false);
      // Focus leaving the panel as it closes goes back to its button.
      if (panel.contains(document.activeElement)) {
        button?.focus();
      }
    };

    window.addEventListener('keydown', onKeyDown);

    return () => window.removeEventListener('keydown', onKeyDown);
  }, [button]);

  const group = <T extends string>(
    key: string,
    title: string,
    values: readonly T[],
    labels: Record<T, string>,
    selected: T | null,
    onChange: (value: T) => void,
    disabled: (value: T) => boolean = () => false
  ) => {
    const labelId = `${groupId}-${key}`;

    return (
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.75 }}>
        <Typography id={labelId} variant="caption" color="text.secondary">
          {title}
        </Typography>
        <ToggleButtonGroup
          exclusive
          size="small"
          value={selected}
          onChange={(_, value: T | null) => value !== null && onChange(value)}
          aria-labelledby={labelId}
          sx={{ '& .MuiToggleButton-root': { whiteSpace: 'nowrap' } }}
        >
          {values.map((value) => (
            <ToggleButton
              key={value}
              value={value}
              disabled={disabled(value)}
              title={disabled(value) ? messages.adminOnly : undefined}
            >
              {labels[value]}
            </ToggleButton>
          ))}
        </ToggleButtonGroup>
      </Box>
    );
  };

  return (
    <Box
      // Floats at the hub's bottom-right while the page scrolls, and rests below its last band.
      sx={{
        position: 'sticky',
        bottom: 12,
        alignSelf: 'flex-end',
        mt: 0,
        mr: -2,
        zIndex: 'speedDial',
      }}
    >
      {/* The button alone in the page's opposite mode, so the demo control stands out; the panel keeps the page's. */}
      <ThemeScope theme="pmm" mode="inverse">
        <Box
          // Its own surface keeps the page from showing through the button's translucent hover.
          sx={{
            display: 'flex',
            p: 0,
            borderRadius: '50%',
            bgcolor: 'background.paper',
            boxShadow: 4,
          }}
        >
          <Tooltip title={messages.trigger}>
            <IconButton
              ref={setButton}
              aria-label={messages.trigger}
              aria-haspopup="dialog"
              aria-expanded={open}
              aria-controls={open ? panelId : undefined}
              aria-keyshortcuts="Shift+D"
              onClick={() => setOpen((was) => !was)}
              size="large"
            >
              <TuneOutlined />
            </IconButton>
          </Tooltip>
        </Box>
      </ThemeScope>
      <Popper
        open={open}
        anchorEl={button}
        placement="top-end"
        modifiers={[{ name: 'offset', options: { offset: [0, 8] } }]}
        sx={{ zIndex: 'speedDial' }}
      >
        <ClickAwayListener
          onClickAway={(event) => {
            if (!button?.contains(event.target as Node)) {
              close(false);
            }
          }}
        >
          <Paper
            ref={attachPanel}
            id={panelId}
            role="dialog"
            aria-labelledby={titleId}
            tabIndex={-1}
            elevation={8}
            onKeyDown={(event: ReactKeyboardEvent) => {
              if (event.key === 'Escape') {
                event.stopPropagation();
                close(true);
              }
            }}
            sx={{
              display: 'flex',
              flexDirection: 'column',
              gap: 2,
              p: 2,
              maxWidth: 'calc(100vw - 32px)',
              outline: 'none',
            }}
          >
            <Box
              sx={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                gap: 2,
              }}
            >
              <Typography id={titleId} variant="subtitle2" component="h2">
                {messages.trigger}
              </Typography>
              <Kbd keys={['⇧', 'D']} label="Shift D" />
            </Box>
            {group(
              'scenario',
              messages.groups.scenario,
              SCENARIOS,
              messages.scenario,
              (SCENARIOS as readonly string[]).includes(params.scenario)
                ? (params.scenario as (typeof SCENARIOS)[number])
                : null,
              (scenario) => setParams({ scenario })
            )}
            {group(
              'freshness',
              messages.groups.freshness,
              FRESHNESS,
              messages.freshness,
              params.freshness,
              // Live drops the simulation and returns to the real 30s refresh.
              (freshness) =>
                setParams(
                  freshness === 'live'
                    ? { freshness, refresh: DEFAULT_HUB_PARAMS.refresh }
                    : { freshness }
                )
            )}
            {group(
              'operations',
              messages.groups.operations(name),
              OPERATIONS,
              messages.operations,
              operations ? 'on' : 'off',
              (ops) => setParams({ ops }),
              (value) => value === 'on' && !admin
            )}
          </Paper>
        </ClickAwayListener>
      </Popper>
    </Box>
  );
};
