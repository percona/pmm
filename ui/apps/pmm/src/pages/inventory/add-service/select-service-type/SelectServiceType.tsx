import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardActionArea from '@mui/material/CardActionArea';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useMemo, type FC } from 'react';
import { Link } from 'react-router-dom';
import { Messages as PageMessages } from '../AddServicePage.messages';
import { AddServiceType } from '../AddServicePage.types';
import { getAddServicePath } from '../AddServicePage.utils';
import { SERVICE_TYPE_CARDS } from './SelectServiceType.constants';
import { Messages } from './SelectServiceType.messages';
import type { SelectServiceTypeProps } from './SelectServiceType.types';

export const SelectServiceType: FC<SelectServiceTypeProps> = ({
  showAzure,
}) => {
  const serviceTypes = useMemo(
    () =>
      SERVICE_TYPE_CARDS.filter(
        ({ type }) => showAzure || type !== AddServiceType.azure
      ),
    [showAzure]
  );

  return (
    <Stack gap={3}>
      <Typography color="text.secondary">{Messages.description}</Typography>
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: { xs: '1fr', md: 'repeat(2, 1fr)' },
          gap: 2,
        }}
      >
        {serviceTypes.map(({ type, icon: Icon }) => (
          <Card key={type} variant="outlined" data-testid={`${type}-instance`}>
            <CardActionArea
              component={Link}
              to={getAddServicePath(type)}
              sx={{ p: 2, height: '100%' }}
            >
              <Stack direction="row" alignItems="center" gap={2}>
                <Icon fontSize="large" />
                <Stack>
                  <Typography variant="h6">
                    {PageMessages.serviceTypes[type]}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    {Messages.cardDescription}
                  </Typography>
                </Stack>
              </Stack>
            </CardActionArea>
          </Card>
        ))}
      </Box>
    </Stack>
  );
};
