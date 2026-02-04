import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Icon } from 'components/icon';
import { FC, useMemo } from 'react';
import { Messages } from './FetchingIndicator.messages';
import { keyframes, useTheme } from '@mui/material/styles';

interface Props {
  isFetching: boolean;
}

const FetchingIndicator: FC<Props> = ({ isFetching }) => {
  const theme = useTheme();
  const styles = useMemo(
    () => ({
      color: isFetching
        ? theme.palette.primary.main
        : theme.palette.text.secondary,
      fontFamily: 'Poppins',
      fontWeight: 600,
      fontSize: 13,
    }),
    [theme, isFetching]
  );
  const fadeInOut = keyframes`
  0% { opacity: 0; }
  50% { opacity: 1; transform: scale(1.1); }
  100% { opacity: 0; }
`;

  if (isFetching) {
    return (
      <Stack
        direction="row"
        alignItems="center"
        data-testid="fetching-indicator-on"
      >
        <Icon
          name="electric-bolt"
          color="primary"
          sx={{ animation: `${fadeInOut} 1.5s infinite` }}
        />
        <Typography color="primary" sx={styles}>
          {Messages.fetching}
        </Typography>
      </Stack>
    );
  }

  return (
    <Stack
      direction="row"
      alignItems="center"
      data-testid="fetching-indicator-off"
    >
      <Icon name="electric-bolt-off" />
      <Typography sx={styles}>{Messages.fetching}</Typography>
    </Stack>
  );
};

export default FetchingIndicator;
