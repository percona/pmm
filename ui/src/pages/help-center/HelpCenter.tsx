import { Button, CardContent, Typography, Box, Card } from '@mui/material';
import {
  Support,
  ForumOutlined,
  DatasetOutlined,
  NorthEast,
  SaveAlt,
  MapOutlined,
} from '@mui/icons-material';
import { KnowledgeBaseIcon } from 'icons';
import { Page } from 'components/page';
import { FC, ReactNode, useCallback } from 'react';
import { Messages } from './HelpCenter.messages';
import { CardsData, cardIds, startIcon } from './HelpCenter.constants';
import { useUser } from 'contexts/user';

export const HelpCenter: FC = () => {
  const { user } = useUser();

  const getIcon = useCallback((cardId: string): ReactNode => {
    switch (cardId) {
      case cardIds.pmmDocs:
        return <KnowledgeBaseIcon />;
      case cardIds.support:
        return <Support />;
      case cardIds.forum:
        return <ForumOutlined />;
      case cardIds.pmmDump:
        return <DatasetOutlined />;
      default:
        return null;
    }
  }, []);

  const getButtonStartIcon = useCallback((iconName: string): ReactNode => {
    switch (iconName) {
      case startIcon.download:
        return <SaveAlt />;
      case startIcon.map:
        return <MapOutlined />;
      default:
        return null;
    }
  }, []);

  const shouldDisplayCard = useCallback(
    (itemId: string): boolean =>
      !(
        !user?.isPMMAdmin &&
        (itemId === cardIds.pmmDump || itemId === cardIds.pmmLogs)
      ),
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
        {CardsData.map(
          (item) =>
            shouldDisplayCard(item.id) && (
              <Card
                style={{
                  borderTop: item.borderColor
                    ? `solid 12px ${item.borderColor}`
                    : 'none',
                }}
              >
                <CardContent style={{ paddingRight: 16, paddingLeft: 16 }}>
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'flex-start',
                      alignItems: 'center',
                      marginBottom: 8,
                    }}
                  >
                    {getIcon(item.id)}
                    <Typography
                      variant="h6"
                      style={{ marginLeft: getIcon(item.id) ? 8 : 0 }}
                    >
                      {item.title}
                    </Typography>
                  </div>

                  <Typography>{item.description}</Typography>
                  <div style={{ display: 'flex', paddingTop: 16 }}>
                    {item.buttons.map((button) => (
                      <Button
                        variant="outlined"
                        size="small"
                        style={{ marginRight: 8 }}
                        startIcon={getButtonStartIcon(button.startIconName)}
                        endIcon={button.target && <NorthEast />}
                        onClick={() => {
                          if (button.target !== '') {
                            window.open(
                              button.url,
                              button.target,
                              'noopener,noreferrer'
                            );
                          } else {
                            window.location.assign(button.url);
                          }
                        }}
                      >
                        {button.buttonText}
                      </Button>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )
        )}
      </Box>
    </Page>
  );
};
