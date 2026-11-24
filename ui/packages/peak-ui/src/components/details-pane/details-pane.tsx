import { forwardRef, useEffect, useRef, useState, type ReactElement } from 'react';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Slide from '@mui/material/Slide';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import KeyboardArrowDownOutlined from '@mui/icons-material/KeyboardArrowDownOutlined';
import KeyboardArrowUpOutlined from '@mui/icons-material/KeyboardArrowUpOutlined';
import type { SxProps, Theme } from '@mui/material/styles';
import { unstable_useEnhancedEffect as useEnhancedEffect, useForkRef } from '@mui/material/utils';
import { BottomPanelCloseIcon } from '../icons';
import Tooltip from '../tooltip';
import type {
  DetailsPaneButtonProps,
  DetailsPaneLabels,
  DetailsPaneProps,
} from './details-pane.types';

const DEFAULT_LABELS: DetailsPaneLabels = {
  previous: 'Previous',
  next: 'Next',
  close: 'Close details',
};

const toSxArray = (sx: SxProps<Theme> | undefined) => (Array.isArray(sx) ? sx : [sx]);

interface HeaderButtonProps {
  /** Tooltip and accessible name. */
  label: string;
  /** Icon inside the button. */
  icon: ReactElement;
  /** Called when the user clicks the button. */
  onClick: () => void;
  /** Whether the button is unavailable. */
  disabled?: boolean;
  /** Props from the pane's matching `*ButtonProps`. */
  buttonProps?: DetailsPaneButtonProps;
}

const HeaderButton = ({ label, icon, onClick, disabled, buttonProps }: HeaderButtonProps) => (
  <Tooltip title={label} arrow describeChild>
    {/* A disabled button fires no events, so the tooltip listens on this wrapper. */}
    <span>
      <IconButton aria-label={label} {...buttonProps} onClick={onClick} disabled={disabled}>
        {icon}
      </IconButton>
    </span>
  </Tooltip>
);

const DetailsPane = forwardRef<HTMLDivElement, DetailsPaneProps>(
  (
    {
      open,
      onClose,
      tabs,
      tab,
      defaultTab,
      onTabChange,
      navigation,
      labels,
      previousButtonProps,
      nextButtonProps,
      closeButtonProps,
      contentProps,
      slideProps,
      sx,
      ...rest
    },
    ref
  ) => {
    const [ownTab, setOwnTab] = useState<string>();
    const [wasOpen, setWasOpen] = useState(open);
    if (wasOpen !== open) {
      setWasOpen(open);
      if (!open) {
        setOwnTab(undefined);
      }
    }
    const { previous, next, close } = { ...DEFAULT_LABELS, ...labels };
    const paneRef = useRef<HTMLDivElement>(null);
    const handleRef = useForkRef(paneRef, ref);
    const returnFocusRef = useRef<HTMLElement | null>(null);

    const requestedTab = tab ?? ownTab ?? defaultTab;
    const selectedTab = tabs.find((item) => item.value === requestedTab) ?? tabs[0];

    useEnhancedEffect(() => {
      const pane = paneRef.current;
      const doc = pane?.ownerDocument ?? document;
      const focused = doc.activeElement instanceof HTMLElement ? doc.activeElement : null;
      if (open) {
        if (!pane?.contains(focused)) {
          returnFocusRef.current = focused;
        }
        return;
      }
      const returnTo = returnFocusRef.current;
      returnFocusRef.current = null;
      if (!pane || !focused || !pane.contains(focused)) {
        return;
      }
      // Focus must leave the pane before it is hidden from assistive technology.
      if (returnTo?.isConnected && returnTo !== doc.body) {
        returnTo.focus();
      }
      if (pane.contains(doc.activeElement)) {
        focused.blur();
      }
    }, [open]);

    useEffect(() => {
      if (!open) {
        return undefined;
      }
      const handleKeyDown = (event: KeyboardEvent) => {
        if (event.key === 'Escape' && !event.defaultPrevented) {
          onClose();
        }
      };
      document.addEventListener('keydown', handleKeyDown);
      return () => document.removeEventListener('keydown', handleKeyDown);
    }, [open, onClose]);

    const handleTabChange = (value: string) => {
      setOwnTab(value);
      onTabChange?.(value);
    };

    return (
      <Slide direction="up" {...slideProps} in={open}>
        <Paper
          ref={handleRef}
          variant="outlined"
          aria-hidden={!open}
          {...rest}
          sx={[
            (theme) => ({
              position: 'absolute',
              inset: 0,
              zIndex: theme.zIndex.modal,
              display: 'flex',
              flexDirection: 'column',
              overflow: 'hidden',
              px: 2,
              borderBottom: 'none',
              borderRadius: `${theme.shape.borderRadiusMd}px ${theme.shape.borderRadiusMd}px 0 0`,
            }),
            ...toSxArray(sx),
          ]}
        >
          <Stack
            direction="row"
            alignItems="center"
            justifyContent="space-between"
            sx={{ flex: 'none', borderBottom: 1, borderColor: 'divider' }}
          >
            <Tabs
              value={selectedTab?.value ?? false}
              onChange={(_, value) => handleTabChange(value)}
              variant="scrollable"
              scrollButtons="auto"
              sx={{ minWidth: 0 }}
            >
              {tabs.map((item) => (
                <Tab key={item.value} {...item.tabProps} value={item.value} label={item.label} />
              ))}
            </Tabs>
            <Stack direction="row" alignItems="center" sx={{ flex: 'none', gap: 0.5, mr: -1 }}>
              {navigation && (
                <>
                  <HeaderButton
                    label={previous}
                    icon={<KeyboardArrowUpOutlined />}
                    onClick={navigation.previous}
                    disabled={navigation.isFirst}
                    buttonProps={previousButtonProps}
                  />
                  <HeaderButton
                    label={next}
                    icon={<KeyboardArrowDownOutlined />}
                    onClick={navigation.next}
                    disabled={navigation.isLast}
                    buttonProps={nextButtonProps}
                  />
                </>
              )}
              <HeaderButton
                label={close}
                icon={<BottomPanelCloseIcon />}
                onClick={onClose}
                buttonProps={closeButtonProps}
              />
            </Stack>
          </Stack>
          <Box
            {...contentProps}
            sx={[
              {
                flex: 1,
                minHeight: 0,
                display: 'flex',
                flexDirection: 'column',
                overflowY: 'auto',
                overflowX: 'hidden',
                py: 2,
              },
              ...toSxArray(contentProps?.sx),
            ]}
          >
            {selectedTab?.content}
          </Box>
        </Paper>
      </Slide>
    );
  }
);

DetailsPane.displayName = 'DetailsPane';

export default DetailsPane;
