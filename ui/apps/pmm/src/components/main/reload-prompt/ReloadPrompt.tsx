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
 *
 * The dismissal covers the build it was made against, so a later upgrade asks again
 * rather than staying silent for as long as the tab lives.
 */
const ReloadPrompt: FC = () => {
  const { isOutdated, serverVersion, serverBuild, reload } = useVersion();
  const [dismissedBuild, setDismissedBuild] = useState<string | null>(null);

  const dismiss = () => setDismissedBuild(serverBuild);

  if (!isOutdated || dismissedBuild === serverBuild) {
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
              onClick={dismiss}
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
              onClick={dismiss}
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
