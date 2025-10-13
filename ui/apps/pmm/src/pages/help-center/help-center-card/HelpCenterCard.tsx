import { Button, CardContent, Typography, Card, Stack } from '@mui/material';
import Support from '@mui/icons-material/Support';
import ForumOutlined from '@mui/icons-material/ForumOutlined';
import DatasetOutlined from '@mui/icons-material/DatasetOutlined';
import NorthEast from '@mui/icons-material/NorthEast';
import SaveAlt from '@mui/icons-material/SaveAlt';
import MapOutlined from '@mui/icons-material/MapOutlined';
import { FC, ReactNode, useCallback } from 'react';
import { CARD_IDS, START_ICON } from '../HelpCenter.constants';
import { HelpCenterCardProps } from './HelpCenterCard.types';
import { Link } from 'react-router-dom';
import { Icon } from 'components/icon';

export const HelpCenterCard: FC<HelpCenterCardProps> = ({ card }) => {
  const { id, title, borderColor, description, buttons } = card;

  const getIcon = useCallback((cardId: string): ReactNode => {
    switch (cardId) {
      case CARD_IDS.pmmDocs:
        return <Icon name="knowledge-base" />;
      case CARD_IDS.support:
        return <Support />;
      case CARD_IDS.forum:
        return <ForumOutlined />;
      case CARD_IDS.pmmDump:
        return <DatasetOutlined />;
      default:
        return null;
    }
  }, []);

  const getButtonStartIcon = useCallback((iconName?: string): ReactNode => {
    switch (iconName) {
      case START_ICON.download:
        return <SaveAlt />;
      case START_ICON.map:
        return <MapOutlined />;
      default:
        return null;
    }
  }, []);

  return (
    <Card key={id} data-testid={`help-card-${id}`} variant="outlined">
      <CardContent
        sx={{
          px: 2,
          '&:last-child': {
            paddingBottom: 2,
          },
          ...(borderColor && { borderTop: `solid 12px ${borderColor}` }),
        }}
      >
        <Stack
          flexDirection="row"
          justifyContent="flex-start"
          alignItems="center"
          marginBottom={1}
        >
          {getIcon(id)}
          <Typography variant="h6" sx={{ ml: getIcon(id) ? 1 : 0 }}>
            {title}
          </Typography>
        </Stack>
        <Typography>{description}</Typography>
        <Stack paddingTop={2} flexDirection="row">
          {buttons.map((button) => (
            <Button
              key={button.url || button.to || button.text}
              variant="outlined"
              size="small"
              sx={{ mr: 1 }}
              startIcon={getButtonStartIcon(button.startIconName)}
              endIcon={button.target && <NorthEast />}
              {...(button.to
                ? {
                    component: Link,
                    to: button.to,
                  }
                : {
                    component: 'a',
                    target: button.target,
                    rel: 'noopener noreferrer',
                    href: button.url,
                  })}
            >
              {button.text}
            </Button>
          ))}
        </Stack>
      </CardContent>
    </Card>
  );
};
