import TuneOutlined from '@mui/icons-material/TuneOutlined';
import Box from '@mui/material/Box';
import ClickAwayListener from '@mui/material/ClickAwayListener';
import Divider from '@mui/material/Divider';
import IconButton, { type IconButtonProps } from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Popper from '@mui/material/Popper';
import Portal from '@mui/material/Portal';
import { styled } from '@mui/material/styles';
import Typography from '@mui/material/Typography';
import { Kbd, Tooltip } from '@percona/peak-ui';
import {
  type FC,
  forwardRef,
  type KeyboardEvent as ReactKeyboardEvent,
  type ReactNode,
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
} from 'react';
import { ThemeScope } from 'components/theme-scope';

// The panel takes Peak's popover paper override, so it shares the dropdowns' Figma Menu surface: border, radius and shadow.
const PanelSurface = styled(Paper, { name: 'MuiPopover', slot: 'Paper' })({});

/** Where a typed D belongs to the field, not to the shortcut. */
const isTyping = (target: EventTarget | null) =>
  target instanceof HTMLElement &&
  (target.isContentEditable ||
    ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName));

/** The button alone in the page's opposite mode, so the control stands out; its tooltip and panel, outside it, keep the page's. */
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

// Above the sidebar, whose drawer sits at MUI's drawer level.
const LAYER = 'modal';

export interface PrototypeControlsProps {
  /** The button's name and the panel's title, such as "Demo controls". */
  title: string;
  /** The panel's settings, under its title. */
  children: ReactNode;
}

/**
 * Prototype only: a button fixed at the viewport's bottom-right, and a non-modal panel above it that holds a page's demo settings.
 * Shift+D opens and closes it; Escape, a click outside or the button close it.
 */
export const PrototypeControls: FC<PrototypeControlsProps> = ({
  title,
  children,
}) => {
  const panelId = useId();
  const titleId = useId();
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

  return (
    // In the body, so no page container (a CSS container among them) can hold it in place of the viewport.
    <Portal>
      <Box
        sx={{ position: 'fixed', right: 12, bottom: 12, zIndex: LAYER }}
        data-testid="prototype-controls"
      >
        <Tooltip title={title} placement="left">
          <InverseButton
            ref={setButton}
            aria-label={title}
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
      </Box>
      <Popper
        open={open}
        anchorEl={button}
        placement="top-end"
        modifiers={[{ name: 'offset', options: { offset: [0, 8] } }]}
        sx={{ zIndex: LAYER }}
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
                {title}
              </Typography>
              <Kbd keys={['⇧', 'D']} label="Shift D" />
            </Box>
            {/* The rule under the title runs edge to edge, setting the header apart; the rules between settings keep the panel's 16px inset. */}
            <Divider sx={{ mx: -2 }} />
            {children}
          </PanelSurface>
        </ClickAwayListener>
      </Popper>
    </Portal>
  );
};
