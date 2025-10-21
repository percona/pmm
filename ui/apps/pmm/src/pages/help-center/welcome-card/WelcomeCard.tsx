import Card from '@mui/material/Card';
import CardMedia from '@mui/material/CardMedia';
import { FC, useCallback } from 'react';
import WelcomeImage from 'assets/welcome-4x.jpg';
import CardContent from '@mui/material/CardContent';
import Stack from '@mui/material/Stack';
import { Icon } from 'components/icon';
import Typography from '@mui/material/Typography';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import CardActions from '@mui/material/CardActions';
import Button from '@mui/material/Button';
import { Messages } from './WelcomeCard.messages';
import { ADD_SERVICE_LINK, WELCOME_CARD_LIST } from './WelcomeCard.constants';
import ListItemIcon from '@mui/material/ListItemIcon';
import { Link } from 'react-router-dom';
import MapOutlinedIcon from '@mui/icons-material/MapOutlined';
import AddIcon from '@mui/icons-material/Add';
import { useUser } from 'contexts/user';
import { useTour } from 'contexts/tour';
import { useUpdateUserInfo } from 'hooks/api/useUser';
import { useServices } from 'hooks/api/useServices';
import { shouldShowAddService } from './WelcomeCard.utils';

const WelcomeCard: FC = () => {
  const { user } = useUser();
  const { startTour } = useTour();
  const { mutate: updateUserInfo } = useUpdateUserInfo();
  const services = useServices();
  const showAddService = shouldShowAddService(services.data);

  const handleDismiss = useCallback(
    () => updateUserInfo({ productTourCompleted: true }),
    [updateUserInfo]
  );

  if (!user || user.info.productTourCompleted) {
    return null;
  }

  return (
    <Card
      data-testid="welcome-card"
      variant="outlined"
      sx={{
        md: 2,
      }}
    >
      <Stack
        sx={{
          position: 'relative',
        }}
      >
        <CardMedia sx={{ height: 200 }} image={WelcomeImage} />
        <Stack
          sx={{
            inset: 0,
            position: 'absolute',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <Icon
            name="pmm-titled"
            sx={{
              height: 80,
              width: 'auto',
              color: 'common.white',
            }}
          />
        </Stack>
      </Stack>
      <CardContent>
        <Typography variant="h3" mb={1}>
          {Messages.title}
        </Typography>
        <Typography>{Messages.description}</Typography>
        <List>
          {WELCOME_CARD_LIST.map(({ icon: Icon, content }, idx) => (
            <ListItem key={idx}>
              <ListItemIcon
                sx={[
                  {
                    color: 'common.black',
                    minWidth: 'auto',
                    ml: -1,
                    pr: 1.5,
                  },
                  (theme) =>
                    theme.applyStyles('dark', {
                      color: 'common.white',
                    }),
                ]}
              >
                <Icon />
              </ListItemIcon>
              <ListItemText>{content}</ListItemText>
            </ListItem>
          ))}
        </List>
        <Typography variant="h6" mb={1}>
          {Messages.ready}
        </Typography>
        <Typography>
          {user?.isPMMAdmin ? Messages.tour : Messages.tourNonAdmin}
        </Typography>
      </CardContent>
      <CardActions sx={{ pb: 3 }}>
        <Button
          startIcon={<MapOutlinedIcon />}
          variant="contained"
          data-testid="welcome-card-start-tour"
          onClick={() => startTour('product')}
        >
          {Messages.startTour}
        </Button>
        {user?.isPMMAdmin && showAddService && (
          <Button
            startIcon={<AddIcon />}
            component={Link}
            to={ADD_SERVICE_LINK}
            data-testid="welcome-card-add-service"
          >
            {Messages.addService}
          </Button>
        )}
        <Button
          variant="text"
          data-testid="welcome-card-dismiss"
          onClick={handleDismiss}
        >
          {Messages.dismiss}
        </Button>
      </CardActions>
    </Card>
  );
};

export default WelcomeCard;
