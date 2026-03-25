
import Box from '@mui/material/Box';
import { Page } from 'components/page';
import { FC, useMemo } from 'react';
import { Messages } from './HelpCenter.messages';
import { getCardData } from './HelpCenter.constants';
import { useUser } from 'contexts/user';
import { HelpCenterCard } from './help-center-card/HelpCenterCard';
import WelcomeCard from './welcome-card/WelcomeCard';
import { cardClasses } from '@mui/material/Card';

export const HelpCenter: FC = () => {
  const { user } = useUser();
  const cards = useMemo(
    () =>
      getCardData().filter((card) => user?.isPMMAdmin || !card.adminOnly),
    [user]
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

          [`.${cardClasses.root}:last-child`]: {
            gridColumn: '1 / -1',
          },
        }}
      >
        {cards.map((item) => (
          <HelpCenterCard key={item.id} card={item} />
        ))}
      </Box>
    </Page>
  );
};
