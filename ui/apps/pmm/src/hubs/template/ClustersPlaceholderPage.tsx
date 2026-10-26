import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Typography from '@mui/material/Typography';
import { PageHeader } from '@percona/peak-ui';
import type { FC } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Page } from 'components/page';
import { Messages } from './hub.messages';

const M = Messages.clustersPage;

/** Stands in for a hub's Clusters page, today's Fleet improved, while it is in design: it lists the clusters a link asked for in `?clusters=a,b`. */
export const ClustersPlaceholderPage: FC = () => {
  const [searchParams] = useSearchParams();
  const clusters = (searchParams.get('clusters') ?? '')
    .split(',')
    .map((name) => name.trim())
    .filter(Boolean);

  return (
    <Page title={M.title} hideTitle>
      <PageHeader title={M.title} />
      <Box
        sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, pt: 2 }}
        data-testid="clusters-placeholder"
      >
        <Typography color="text.secondary">{M.fleet}</Typography>
        <Typography color="text.secondary">
          {clusters.length ? M.received(clusters.length) : M.none}
        </Typography>
        {clusters.length > 0 && (
          <Box
            component="ul"
            sx={{
              display: 'flex',
              flexWrap: 'wrap',
              gap: 1,
              m: 0,
              p: 0,
              listStyle: 'none',
            }}
          >
            {clusters.map((name) => (
              <li key={name}>
                <Chip label={name} size="small" variant="outlined" />
              </li>
            ))}
          </Box>
        )}
      </Box>
    </Page>
  );
};
