import Stack from '@mui/material/Stack';
import { Chip } from '@percona/peak-ui';
import type { FC } from 'react';
import { toLabelTags } from '../../Inventory.utils';

interface Props {
  labels: Record<string, string>;
}

export const LabelsList: FC<Props> = ({ labels }) => (
  <Stack direction="row" flexWrap="wrap" gap={1}>
    {toLabelTags(labels).map((tag) => (
      <Chip key={tag} size="small" label={tag} />
    ))}
  </Stack>
);
