import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import { Link } from 'react-router-dom';
import { Messages } from './AlertsStatusEmptyState.messages';

const AlertsStatusEmptyState = () => (
  <Stack
    direction="column"
    alignItems="center"
    justifyContent="center"
    gap={1}
    sx={{ width: '100%', py: 6, color: 'text.secondary' }}
  >
    <Typography variant="subtitle1">{Messages.subtitle}</Typography>
    <Typography variant="body2">{Messages.body}</Typography>
    <Button
      size="small"
      variant="outlined"
      startIcon={<AddOutlinedIcon />}
      sx={{ mt: 1 }}
      component={Link}
      to="/graph/alerting/alert-rule-templates"
    >
      {Messages.button}
    </Button>
  </Stack>
);

export default AlertsStatusEmptyState;
