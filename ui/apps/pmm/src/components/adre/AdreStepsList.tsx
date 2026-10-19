import Check from '@mui/icons-material/Check';
import Close from '@mui/icons-material/Close';
import ExpandLess from '@mui/icons-material/ExpandLess';
import ExpandMore from '@mui/icons-material/ExpandMore';
import OpenInNew from '@mui/icons-material/OpenInNew';
import {
  Box,
  Button,
  CircularProgress,
  Collapse,
  Stack,
  Typography,
} from '@mui/material';
import { FC, useMemo, useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { useManagedServices } from 'hooks/api/useServices';
import type { ProgressStep } from 'hooks/useAdreChat';
import { getNativeQanEnabledSnapshot } from 'utils/pmmFeatureFlags';
import { stripQanServiceId } from 'utils/qanServiceId';
import { AdreCodeBlock } from './adre-code-block';
import { stepLink, stepQuery, stepSummary } from './adreToolSteps';

const actionSx = {
  minWidth: 0,
  px: 0.75,
  py: 0.25,
  fontSize: 12,
  fontWeight: 400,
  color: '#CFCBFF',
  '& .MuiButton-startIcon, & .MuiButton-endIcon': { mx: 0.5 },
} as const;

/** One tool step: what it did, the exact query it ran, and the page that shows the same data. */
const StepItem: FC<{
  step: ProgressStep;
  working: boolean;
  serviceName?: string;
}> = ({ step, working, serviceName }) => {
  const [open, setOpen] = useState(false);
  const query = stepQuery(step);
  const link = stepLink(step, getNativeQanEnabledSnapshot(), serviceName);
  const summary = stepSummary(step);
  const running = step.status === 'running' && working;

  return (
    <Stack component="li" direction="row" gap={1.25} sx={{ minWidth: 0 }}>
      <Box sx={{ flex: '0 0 16px', mt: '2px', display: 'flex' }}>
        {running ? (
          <CircularProgress size={14} aria-label="Running" />
        ) : step.failed ? (
          <Close aria-label="Failed" sx={{ fontSize: 16, color: '#FCA5A5' }} />
        ) : (
          <Check aria-label="Done" sx={{ fontSize: 16, color: '#86EFAC' }} />
        )}
      </Box>
      <Stack gap={0.25} sx={{ minWidth: 0, flex: 1 }}>
        <Typography
          sx={{
            fontFamily: 'Roboto Mono, monospace',
            fontSize: 12,
            color: '#D9D7DE',
            overflowWrap: 'anywhere',
          }}
        >
          {step.toolName}
        </Typography>
        {summary && (
          <Typography
            title={summary}
            sx={{
              fontSize: 12,
              color: '#A9A7B0',
              overflowWrap: 'anywhere',
              display: '-webkit-box',
              WebkitLineClamp: 2,
              WebkitBoxOrient: 'vertical',
              overflow: 'hidden',
            }}
          >
            {summary}
          </Typography>
        )}
        {step.error && (
          <Typography
            sx={{ fontSize: 12, color: '#FCA5A5', overflowWrap: 'anywhere' }}
          >
            {step.error}
          </Typography>
        )}
        {(query || link) && (
          <Stack direction="row" flexWrap="wrap" sx={{ ml: -0.75 }}>
            {query && (
              <Button
                size="small"
                onClick={() => setOpen((o) => !o)}
                aria-expanded={open}
                startIcon={open ? <ExpandLess /> : <ExpandMore />}
                sx={actionSx}
              >
                {open ? 'Hide query' : 'Show query'}
              </Button>
            )}
            {link && (
              <Button
                size="small"
                component={RouterLink}
                to={link.to}
                target="_blank"
                rel="noopener"
                endIcon={<OpenInNew sx={{ fontSize: '14px !important' }} />}
                sx={actionSx}
              >
                {link.label}
              </Button>
            )}
          </Stack>
        )}
        {query && (
          <Collapse in={open} unmountOnExit>
            <Box
              sx={{
                mt: -0.5,
                // Models write SQL on one line; wrap it rather than scroll sideways.
                '& pre code': {
                  whiteSpace: 'pre-wrap',
                  overflowWrap: 'anywhere',
                },
              }}
            >
              <AdreCodeBlock>
                <code className={`language-${query.language}`}>
                  {query.text}
                </code>
              </AdreCodeBlock>
            </Box>
          </Collapse>
        )}
      </Stack>
    </Stack>
  );
};

export interface AdreStepsListProps {
  steps: ProgressStep[];
  /** While the answer is written; steps still running show a spinner. */
  working: boolean;
}

/** The tool steps behind an answer, each traceable to its query and data. */
export const AdreStepsList: FC<AdreStepsListProps> = ({ steps, working }) => {
  const serviceId = (s: ProgressStep) =>
    typeof s.params?.service_id === 'string'
      ? stripQanServiceId(s.params.service_id)
      : '';
  const { data } = useManagedServices(
    {},
    { enabled: steps.some(serviceId), staleTime: 60_000 }
  );
  const names = useMemo(
    () => new Map(data?.services.map((s) => [s.serviceId, s.serviceName])),
    [data]
  );

  return (
    <Stack component="ol" gap={1.5} sx={{ m: 0, p: 0, listStyle: 'none' }}>
      {steps.map((step) => (
        <StepItem
          key={step.id}
          step={step}
          working={working}
          serviceName={names.get(serviceId(step))}
        />
      ))}
    </Stack>
  );
};
