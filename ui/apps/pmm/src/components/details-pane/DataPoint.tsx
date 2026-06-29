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
  tooltip?: string;
  size?: GridProps['size'];
}

const DataPoint: FC<Props> = ({ title, tooltip, children, size }) => (
  <Grid
    size={size}
    sx={{
      display: 'flex',
      flexDirection: 'column',
      justifyContent: 'space-between',
    }}
  >
    <Stack direction="row" alignItems="center">
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
    <Box py={1.5}>{children || <UnavailableText />}</Box>
    <Divider />
  </Grid>
);

export default DataPoint;
