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
import { HelpCenterCardProps, CardButton } from './HelpCenterCard.types';

export const HelpCenterCard: FC<HelpCenterCardProps> = (props) => {
  const { shouldDisplayCard, card } = props;

  const { id, title, borderColor, description, buttons, adminOnly } = card;

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

  const getButtonStartIcon = useCallback((iconName: string): ReactNode => {
    switch (iconName) {
      case START_ICON.download:
        return <SaveAlt />;
      case START_ICON.map:
        return <MapOutlined />;
      default:
        return null;
    }
  }, []);

  const onButtonClick = useCallback((button: CardButton) => {
    if (button.target !== '') {
      window.open(button.url, button.target, 'noopener,noreferrer');
    } else {
      window.location.assign(button.url);
    }
  }, []);

  if (!shouldDisplayCard(adminOnly)) {
    return null;
  }

  return (
    <Card
      style={{
        borderTop: borderColor ? `solid 12px ${borderColor}` : 'none',
      }}
      key={id}
      data-testid={`help-card-${id}`}
    >
      <CardContent style={{ paddingRight: 16, paddingLeft: 16 }}>
        <Stack
          flexDirection={'row'}
          justifyContent={'flex-start'}
          alignItems={'center'}
          marginBottom={'8px'}
        >
          {getIcon(id)}
          <Typography variant="h6" style={{ marginLeft: getIcon(id) ? 8 : 0 }}>
            {title}
          </Typography>
        </Stack>

        <Typography>{description}</Typography>
        <Stack paddingTop={'16px'} flexDirection={'row'}>
          {buttons.map((button) => (
            <Button
              key={button.url}
              variant="outlined"
              component="a"
              size="small"
              style={{ marginRight: 8 }}
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
