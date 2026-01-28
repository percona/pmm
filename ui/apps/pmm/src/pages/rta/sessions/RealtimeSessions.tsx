import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { FC } from 'react';
import { Messages } from './RealtimeSessions.messages';
import SessionsTable from './sessions-table/SessionsTable';
import Button from '@mui/material/Button';
import { Link as RouterLink, useSearchParams } from 'react-router-dom';
import Link from '@mui/material/Link';
import ArrowBackOutlinedIcon from '@mui/icons-material/ArrowBackOutlined';
import { DOCS_URLS } from 'lib/constants';
import { RealtimePage } from '../components/rta-page';

const RealtimeSessionsPage: FC = () => {
  const [searchParams] = useSearchParams();

  return (
    <RealtimePage>
      <Stack direction="column" gap={1} sx={{ flexShrink: 0 }}>
        {searchParams.get('fromOverview') && (
          <RouterLink to="/rta/overview">
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
        <Stack sx={{
          display: 'flex',
          flexDirection: {
            xs: 'column',
            md: 'row',
          },
          gap: {
            xs: 1,
            md: 2,
          },
        }}>
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
    </RealtimePage>
  );
};

export default RealtimeSessionsPage;
