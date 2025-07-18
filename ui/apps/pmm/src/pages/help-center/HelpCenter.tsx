import Box from '@mui/material/Box';
import { Page } from 'components/page';
import { FC } from 'react';
import { Messages } from './HelpCenter.messages';
import { CARDS_DATA } from './HelpCenter.constants';
import { useUser } from 'contexts/user';
import { HelpCenterCard } from './help-center-card/HelpCenterCard';
import WelcomeCard from './welcome-card/WelcomeCard';

export const HelpCenter: FC = () => {
  const { user } = useUser();
  const cards = CARDS_DATA.filter(
    (card) => user?.isPMMAdmin || !card.adminOnly
  );

  return (
    <Page topBar={<WelcomeCard />} title={Messages.pageTitle}>
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
        {cards.map((item) => (
          <HelpCenterCard key={item.id} card={item} />
        ))}
      </Box>
    </Page>
  );
};
