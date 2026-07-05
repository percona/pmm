import { FC, ReactNode, useEffect, useState } from 'react';
import CloseIcon from '@mui/icons-material/Close';
import CloseFullscreenOutlinedIcon from '@mui/icons-material/CloseFullscreenOutlined';
import OpenInFullOutlinedIcon from '@mui/icons-material/OpenInFullOutlined';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Divider from '@mui/material/Divider';
import Drawer from '@mui/material/Drawer';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { CopyToClipboardButton } from '@percona/percona-ui';
import { format } from 'date-fns';
import {
  DRAWER_CLOSED_WIDTH,
  DRAWER_WIDTH,
} from 'components/sidebar/drawer/Drawer.constants';
import { useNavigation } from 'contexts/navigation/navigation.hooks';
import { CheckResultHistoryItem } from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import {
  ADVISOR_INTERVAL,
  ADVISOR_RESULT_STATUS,
  SEVERITY,
  TIME_FORMAT,
} from 'lib/constants';
import { capitalize } from 'utils/text.utils';
import { Messages } from '../AdvisorInsights.messages';

const EM_DASH = '—';

const SEVERITY_CHIP_COLOR: Record<
  Severity,
  'error' | 'warning' | 'info' | 'default'
> = {
  [Severity.emergency]: 'error',
  [Severity.alert]: 'error',
  [Severity.critical]: 'error',
  [Severity.error]: 'error',
  [Severity.warning]: 'warning',
  [Severity.notice]: 'info',
  [Severity.info]: 'info',
  [Severity.debug]: 'default',
  [Severity.unspecified]: 'default',
};

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
    data-testid={`details-field-${label.toLowerCase().replace(/[^a-z]+/g, '-')}`}
  >
    <Typography variant="caption" color="text.secondary">
      {label}
    </Typography>
    <Box sx={{ flex: 1 }}>{children}</Box>
    <Divider />
  </Stack>
);

interface InsightDetailsPaneProps {
  insight: CheckResultHistoryItem | null;
  // enabled state of the underlying check; undefined when unknown
  checkEnabled?: boolean;
  onClose: () => void;
}

export const InsightDetailsPane: FC<InsightDetailsPaneProps> = ({
  insight,
  checkEnabled,
  onClose,
}) => {
  const [maximized, setMaximized] = useState(false);
  const { navOpen } = useNavigation();
  const open = !!insight;
  // when maximized, take all the space except the main navigation
  const sidebarWidth = navOpen ? DRAWER_WIDTH : DRAWER_CLOSED_WIDTH;

  // start each viewing at the default height
  useEffect(() => {
    if (!open) {
      setMaximized(false);
    }
  }, [open]);

  const m = Messages.details;
  const labels = Object.entries(insight?.labels ?? {});

  return (
    <Drawer
      anchor="bottom"
      open={open}
      onClose={onClose}
      slotProps={{
        paper: {
          // @ts-expect-error data-testid is passed through to the DOM
          'data-testid': 'insight-details-pane',
          sx: {
            height: maximized ? '100vh' : '60vh',
            left: { xs: 0, md: maximized ? sidebarWidth : 0 },
            p: 2,
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
            data-testid="insight-details-maximize"
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
            data-testid="insight-details-close"
          >
            <CloseIcon fontSize="small" />
          </IconButton>
        </Stack>
      </Stack>
      {insight && (
        <Box sx={{ overflow: 'auto', mt: 2 }}>
          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: 'repeat(4, 1fr)',
              columnGap: 4,
              rowGap: 3,
            }}
          >
            <Field label={m.insight}>
              <Typography variant="body1">{insight.summary}</Typography>
            </Field>
            <Field label={m.severity}>
              <Chip
                size="small"
                color={SEVERITY_CHIP_COLOR[insight.severity]}
                label={SEVERITY[insight.severity]}
              />
            </Field>
            <Field label={m.status}>
              <Typography variant="body1">
                {ADVISOR_RESULT_STATUS[insight.status]}
              </Typography>
            </Field>
            <Field label={m.advisorStatus}>
              <Typography variant="body1">
                {checkEnabled === undefined
                  ? EM_DASH
                  : checkEnabled
                    ? m.enabled
                    : m.disabled}
              </Typography>
            </Field>

            <Field label={m.description} span={4}>
              <Typography variant="body1">
                {insight.description || EM_DASH}
              </Typography>
            </Field>

            <Field label={m.readMore} span={4}>
              {insight.readMoreUrl ? (
                <Link
                  href={insight.readMoreUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  variant="body1"
                >
                  {insight.readMoreUrl}
                </Link>
              ) : (
                <Typography variant="body1">{EM_DASH}</Typography>
              )}
            </Field>

            <Field label={m.service}>
              <Stack direction="row" alignItems="center" gap={0.5}>
                <Typography variant="body1">{insight.serviceName}</Typography>
                <CopyToClipboardButton textToCopy={insight.serviceId} />
              </Stack>
            </Field>
            <Field label={m.node}>
              <Stack direction="row" alignItems="center" gap={0.5}>
                <Typography variant="body1">{insight.nodeName}</Typography>
                <CopyToClipboardButton textToCopy={insight.nodeId} />
              </Stack>
            </Field>
            <Field label={m.category}>
              <Typography variant="body1">
                {capitalize(insight.category)}
              </Typography>
            </Field>
            <Field label={m.environment}>
              <Typography variant="body1">{EM_DASH}</Typography>
            </Field>

            <Field label={m.cluster}>
              <Typography variant="body1">{EM_DASH}</Typography>
            </Field>
            <Field label={m.replicationSet}>
              <Typography variant="body1">{EM_DASH}</Typography>
            </Field>
            <Field label={m.region}>
              <Typography variant="body1">{EM_DASH}</Typography>
            </Field>
            <Field label={m.az}>
              <Typography variant="body1">{EM_DASH}</Typography>
            </Field>

            <Field label={m.checkedAt}>
              <Typography variant="body1">
                {insight.checkedAt
                  ? format(new Date(insight.checkedAt), TIME_FORMAT)
                  : EM_DASH}
              </Typography>
            </Field>
            <Field label={m.checkInterval}>
              <Typography variant="body1">
                {ADVISOR_INTERVAL[insight.interval]}
              </Typography>
            </Field>
            <Field label={m.firstDetected}>
              <Typography variant="body1">{EM_DASH}</Typography>
            </Field>
            <Field label={m.checkName}>
              <Typography variant="body1">
                {insight.advisorName
                  ? `${insight.advisorName}/${insight.checkName}`
                  : insight.checkName}
              </Typography>
            </Field>
          </Box>

          {labels.length > 0 && (
            <Stack gap={1} sx={{ mt: 3 }}>
              <Typography variant="h6">{m.otherLabels}</Typography>
              <Stack direction="row" flexWrap="wrap" gap={1}>
                {labels.map(([key, value]) => (
                  <Chip key={key} size="small" label={`${key}: ${value}`} />
                ))}
              </Stack>
            </Stack>
          )}
        </Box>
      )}
    </Drawer>
  );
};
