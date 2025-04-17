import { Box } from '@mui/material';
import { Page } from 'components/page';
import { FC, useCallback } from 'react';
import { Messages } from './HelpCenter.messages';
import { CardsData } from './HelpCenter.constants';
import { useUser } from 'contexts/user';
import { HelpCenterCard } from './help-center-card/HelpCenterCard';

export const HelpCenter: FC = () => {
  const { user } = useUser();

  const shouldDisplayCard = useCallback(
    (adminOnly: boolean): boolean => !(!user?.isPMMAdmin && adminOnly),
    [user]
  );

  return (
    <Page title={Messages.pageTitle}>
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: {
            xs: '1fr',
            sm: '1fr',
            md: 'repeat(2, 1fr)',
          },
          gap: 4,
        }}
      >
        {CardsData.map((item) => (
          <HelpCenterCard card={item} shouldDisplayCard={shouldDisplayCard} />
        ))}
      </Box>
    </Page>
  );
};
