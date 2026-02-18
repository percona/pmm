import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import UnavailableText from 'components/unavailable-text';
import { FC } from 'react';

type Props = {
  mainText?: string;
  subText?: string;
  size?: 'small' | 'medium';
};

const BigNumberMetric: FC<Props> = ({ mainText, subText, size = 'medium' }) => (
  <Box style={{ display: 'flex', alignItems: 'baseline' }}>
    {mainText ? (
      <Typography
        variant={size === 'small' ? 'body1' : 'h5'}
        fontWeight="600"
        fontFamily="Roboto Mono, monospace"
        overflow="hidden"
        textOverflow="ellipsis"
        whiteSpace="nowrap"
      >
        {mainText}
      </Typography>
    ) : (
      <UnavailableText />
    )}
    {subText && (
      <Typography
        variant="body2"
        fontWeight="400"
        fontFamily="Roboto Mono, monospace"
        ml={0.5}
      >
        {subText}
      </Typography>
    )}
  </Box>
);

export default BigNumberMetric;
