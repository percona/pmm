import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Tooltip } from '@percona/peak-ui';
import { PMM_DOCS_INVENTORY_SERVICES_URL } from 'lib/constants';
import type { FC } from 'react';
import { Messages as InventoryMessages } from '../../Inventory.messages';
import { Messages } from '../Services.messages';

const AddressInfo: FC = () => (
  <Typography variant="body2" py={0.5}>
    {Messages.addressInfo}{' '}
    <Link
      color="inherit"
      href={PMM_DOCS_INVENTORY_SERVICES_URL}
      target="_blank"
      rel="noopener noreferrer"
      data-testid="address-info-docs-link"
    >
      {Messages.learnMore}
    </Link>
  </Typography>
);

export const AddressHeader: FC = () => (
  <Stack direction="row" alignItems="center" gap={0.5}>
    {InventoryMessages.columns.address}
    <Tooltip title={<AddressInfo />}>
      <InfoOutlinedIcon
        fontSize="small"
        color="action"
        tabIndex={0}
        // The header label sorts the column on click
        onClick={(event) => event.stopPropagation()}
        data-testid="address-info-icon"
      />
    </Tooltip>
  </Stack>
);
