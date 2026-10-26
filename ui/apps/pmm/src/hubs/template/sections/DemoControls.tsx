import TuneOutlined from '@mui/icons-material/TuneOutlined';
import Box from '@mui/material/Box';
import ClickAwayListener from '@mui/material/ClickAwayListener';
import Divider from '@mui/material/Divider';
import FormControlLabel from '@mui/material/FormControlLabel';
import IconButton, { type IconButtonProps } from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Popper from '@mui/material/Popper';
import { styled } from '@mui/material/styles';
import Switch from '@mui/material/Switch';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import Typography from '@mui/material/Typography';
import { Kbd, Tooltip } from '@percona/peak-ui';
import {
  type FC,
  forwardRef,
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

// The panel takes Peak's popover paper override, so it shares the dropdowns' Figma Menu surface: border, radius and shadow.
const PanelSurface = styled(Paper, { name: 'MuiPopover', slot: 'Paper' })({});

/** Where a typed D belongs to the field, not to the shortcut. */
const isTyping = (target: EventTarget | null) =>
  target instanceof HTMLElement &&
  (target.isContentEditable ||
    ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName));

/** The button alone in the page's opposite mode, so the demo control stands out; its tooltip and panel, outside it, keep the page's. */
const InverseButton = forwardRef<HTMLButtonElement, IconButtonProps>(
  (props, ref) => (
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
        <IconButton ref={ref} {...props} />
      </Box>
    </ThemeScope>
  )
);

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
  const scenarioLabelId = useId();
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

  // Label left, switch right; the switch's own padding reaches into the panel's, as its track lines up with the edge.
  // The row keeps only the label's height, so the 38px switch doesn't add its padding to the space around the rules.
  const switchRow = {
    ml: 0,
    mr: -1.5,
    my: -1,
    gap: 2,
    justifyContent: 'space-between',
  };
  // The rule under the title runs edge to edge, setting the header apart; the rules between settings keep the panel's 16px inset.

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
      <Tooltip title={messages.trigger}>
        <InverseButton
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
        </InverseButton>
      </Tooltip>
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
          <PanelSurface
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
              // The same space above and below every rule.
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
              <Typography id={titleId} variant="sectionHeading" component="h2">
                {messages.trigger}
              </Typography>
              <Kbd keys={['⇧', 'D']} label="Shift D" />
            </Box>
            <Divider sx={{ mx: -2 }} />
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
              {/* As strong as the switches' labels, which name settings the same way. */}
              <Typography id={scenarioLabelId}>
                {messages.scenarioLabel}
              </Typography>
              <ToggleButtonGroup
                exclusive
                size="small"
                value={
                  (SCENARIOS as readonly string[]).includes(params.scenario)
                    ? params.scenario
                    : null
                }
                onChange={(_, scenario: (typeof SCENARIOS)[number] | null) =>
                  scenario !== null && setParams({ scenario })
                }
                aria-labelledby={scenarioLabelId}
                sx={{ '& .MuiToggleButton-root': { whiteSpace: 'nowrap' } }}
              >
                {SCENARIOS.map((scenario) => (
                  <ToggleButton key={scenario} value={scenario}>
                    {messages.scenario[scenario]}
                  </ToggleButton>
                ))}
              </ToggleButtonGroup>
            </Box>
            <Divider />
            <FormControlLabel
              labelPlacement="start"
              label={messages.stale}
              sx={switchRow}
              control={
                <Switch
                  checked={params.freshness === 'stale'}
                  // Off drops the simulation and returns to the real 30s refresh.
                  onChange={(_, stale) =>
                    setParams(
                      stale
                        ? { freshness: 'stale' }
                        : {
                            freshness: 'live',
                            refresh: DEFAULT_HUB_PARAMS.refresh,
                          }
                    )
                  }
                />
              }
            />
            <Divider />
            <FormControlLabel
              labelPlacement="start"
              label={messages.operations(name)}
              disabled={!admin}
              title={admin ? undefined : messages.adminOnly}
              sx={switchRow}
              control={
                <Switch
                  checked={operations}
                  onChange={(_, on) => setParams({ ops: on ? 'on' : 'off' })}
                />
              }
            />
          </PanelSurface>
        </ClickAwayListener>
      </Popper>
    </Box>
  );
};
