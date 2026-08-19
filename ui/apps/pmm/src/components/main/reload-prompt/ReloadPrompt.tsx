import Button from '@mui/material/Button';
import Card from '@mui/material/Card';
import IconButton from '@mui/material/IconButton';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import CloseIcon from '@mui/icons-material/Close';
import { useVersion } from 'contexts/version';
import { FC, useState } from 'react';
import { Messages } from './ReloadPrompt.messages';

/**
 * Asks before reloading a tab the user is looking at, so an external upgrade cannot
 * discard whatever they are in the middle of. Dismissing only hides the prompt: the
 * tab still reloads on its own once it is hidden.
 */
const ReloadPrompt: FC = () => {
  const { isOutdated, serverVersion, reload } = useVersion();
  const [dismissed, setDismissed] = useState(false);

  if (!isOutdated || dismissed) {
    return null;
  }

  return (
    <Snackbar open anchorOrigin={{ horizontal: 'right', vertical: 'bottom' }}>
      <Card
        elevation={12}
        sx={{
          width: 500,
          p: 2,
        }}
        data-testid="reload-prompt"
      >
        <Stack gap={2}>
          <Stack direction="row">
            <Typography>
              <Typography
                component="span"
                fontWeight="bold"
                display="inline-block"
                data-testid="reload-prompt-title"
              >
                {Messages.title(serverVersion)}
              </Typography>
              <span data-testid="reload-prompt-description">
                {Messages.description}
              </span>
            </Typography>
            <IconButton
              data-testid="reload-prompt-close-button"
              onClick={() => setDismissed(true)}
              sx={{
                alignSelf: 'flex-start',
              }}
            >
              <CloseIcon sx={{ width: 20, height: 20 }} />
            </IconButton>
          </Stack>
          <Stack gap={1} direction="row">
            <Button
              variant="contained"
              onClick={reload}
              data-testid="reload-prompt-reload-button"
            >
              {Messages.reload}
            </Button>
            <Button
              onClick={() => setDismissed(true)}
              data-testid="reload-prompt-dismiss-button"
            >
              {Messages.dismiss}
            </Button>
          </Stack>
        </Stack>
      </Card>
    </Snackbar>
  );
};

export default ReloadPrompt;
