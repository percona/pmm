import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { FC } from 'react';
import { Messages } from './RealTimeSessions.messages';
import SessionsTable from './sessions-table/SessionsTable';
import Button from '@mui/material/Button';
import { Link as RouterLink, useParams } from 'react-router-dom';
import Link from '@mui/material/Link';
import ArrowBackOutlinedIcon from '@mui/icons-material/ArrowBackOutlined';
import { DOCS_URLS } from 'lib/constants';

const RealTimeSessionsPage: FC = () => {
  const { fromAnalysis } = useParams();

  return (
    <Stack
      direction="column"
      gap={2}
      p={2}
      sx={{
        height: '100%',
        maxHeight: 'calc(100vh - 64px)', // Account for header height
        overflow: 'hidden',
        display: 'flex',
      }}
    >
      <Stack direction="column" gap={1} sx={{ flexShrink: 0 }}>
        {fromAnalysis && (
          <RouterLink to="/rta/live">
            <Button
              size="small"
              startIcon={<ArrowBackOutlinedIcon />}
              sx={{
                color: 'text.primary',
              }}
            >
              {Messages.backToAnalysis}
            </Button>
          </RouterLink>
        )}
        <Typography variant="h6">{Messages.pageTitle}</Typography>
        <Typography variant="body2">{Messages.pageDescription}</Typography>
        <Stack direction="row" justifyContent="flex-start" gap={2}>
          <Link
            variant="body2"
            href={DOCS_URLS.qan}
            rel="noopener noreferrer"
            target="_blank"
          >
            {Messages.documentation}
          </Link>
          <Link
            variant="body2"
            href={DOCS_URLS.forums}
            rel="noopener noreferrer"
            target="_blank"
          >
            {Messages.provideFeedback}
          </Link>
        </Stack>
      </Stack>
      <Stack
        sx={{
          flex: 1,
          minHeight: 0,
          overflow: 'hidden',
          height: 0, // Force flex item to respect parent height
        }}
      >
        <SessionsTable />
      </Stack>
    </Stack>
  );
};

export default RealTimeSessionsPage;
