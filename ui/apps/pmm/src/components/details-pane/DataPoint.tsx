import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Tooltip } from '@percona/percona-ui';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { FC, PropsWithChildren } from 'react';
import Box from '@mui/material/Box';
import Divider from '@mui/material/Divider';
import Grid, { GridProps } from '@mui/material/Grid';
import UnavailableText from 'components/unavailable-text';

interface Props extends PropsWithChildren {
  title: string;
  subtitle?: string;
  tooltip?: string;
  // When set, the data point renders as a Grid item; otherwise as a plain Stack
  // (for callers that do their own grid layout).
  size?: GridProps['size'];
}

const DataPoint: FC<Props> = ({ title, subtitle, tooltip, size, children }) => {
  const content = (
    <>
      <span>
        <Stack direction="row" alignItems="center" gap={0.5}>
          <Typography variant="body1" fontFamily="Poppins" fontWeight="600">
            {title}
          </Typography>
          {tooltip && (
            <Tooltip title={tooltip} arrow>
              <IconButton
                size="small"
                sx={{ color: 'text.secondary' }}
                aria-label={tooltip}
              >
                <InfoOutlinedIcon />
              </IconButton>
            </Tooltip>
          )}
        </Stack>
        {subtitle && (
          <Typography
            variant="body2"
            fontFamily="Roboto Mono, monospace"
            fontWeight="400"
            color="text.disabled"
            ml={1}
          >
            {subtitle}
          </Typography>
        )}
      </span>
      <Box py={1.5}>{children || <UnavailableText />}</Box>
      <Divider sx={{ mt: 'auto' }} />
    </>
  );

  return size !== undefined ? (
    <Grid
      size={size}
      sx={{
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'space-between',
      }}
    >
      {content}
    </Grid>
  ) : (
    <Stack height="100%">{content}</Stack>
  );
};

export default DataPoint;
