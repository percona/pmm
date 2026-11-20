import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import type { FC } from 'react';
import type { DiscoveryDocsProps } from './DiscoveryDocs.types';

export const DiscoveryDocs: FC<DiscoveryDocsProps> = ({ links }) => (
  <Stack gap={1} data-testid="discovery-docs">
    {links.map(({ label, url }) => (
      <Link key={url} href={url} target="_blank" rel="noopener noreferrer">
        {label}
      </Link>
    ))}
  </Stack>
);
