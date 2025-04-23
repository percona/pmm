import { Button, CardContent, Typography, Card, Stack } from '@mui/material';
import {
  Support,
  ForumOutlined,
  DatasetOutlined,
  NorthEast,
  SaveAlt,
  MapOutlined,
} from '@mui/icons-material';
import { KnowledgeBaseIcon } from 'icons';
import { FC, ReactNode, useCallback } from 'react';
import { CARD_IDS, START_ICON } from '../HelpCenter.constants';
import { HelpCenterCardProps, HelpCardButton } from './HelpCenterCard.types';

export const HelpCenterCard: FC<HelpCenterCardProps> = ({ card }) => {
  const { id, title, borderColor, description, buttons } = card;

  const getIcon = useCallback((cardId: string): ReactNode => {
    switch (cardId) {
      case CARD_IDS.pmmDocs:
        return <KnowledgeBaseIcon />;
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

  const onButtonClick = useCallback((button: HelpCardButton) => {
    if (button.target) {
      window.open(button.url, button.target, 'noopener,noreferrer');
    } else {
      //TO DO: use react router once the grafana iframe is in place
      button.url && window.location.assign(button.url);
    }
  }, []);

  return (
    <Card
      sx={{
        borderTop: borderColor ? `solid 12px ${borderColor}` : 'none',
      }}
      key={id}
      data-testid={`help-card-${id}`}
    >
      <CardContent sx={{ px: 2 }}>
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
              key={button.url}
              variant="outlined"
              component="a"
              size="small"
              sx={{ mr: 1 }}
              startIcon={getButtonStartIcon(button.startIconName)}
              endIcon={button.target && <NorthEast />}
              onClick={() => onButtonClick(button)}
            >
              {button.text}
            </Button>
          ))}
        </Stack>
      </CardContent>
    </Card>
  );
};
