import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import { Chip } from '@percona/peak-ui';
import type { FC } from 'react';
import { MAX_VISIBLE_SERVICES } from '../../DumpPage.constants';
import { Messages } from '../../DumpPage.messages';

interface Props {
  serviceNames: string[];
}

export const DumpServices: FC<Props> = ({ serviceNames }) => {
  if (!serviceNames.length) {
    return <Chip size="small" label={Messages.allServices} />;
  }

  const visible = serviceNames.slice(0, MAX_VISIBLE_SERVICES);
  const hidden = serviceNames.slice(MAX_VISIBLE_SERVICES);

  return (
    <Stack direction="row" flexWrap="wrap" gap={0.5}>
      {visible.map((name) => (
        <Chip key={name} size="small" label={name} />
      ))}
      {!!hidden.length && (
        <Tooltip
          title={
            <Stack>
              {hidden.map((name) => (
                <span key={name}>{name}</span>
              ))}
            </Stack>
          }
        >
          <Box component="span">
            <Chip size="small" label={Messages.moreServices(hidden.length)} />
          </Box>
        </Tooltip>
      )}
    </Stack>
  );
};
