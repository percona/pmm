import Box from '@mui/material/Box';
import Typography, { TypographyProps } from '@mui/material/Typography';
import UnavailableText from 'components/unavailable-text';
import { FC } from 'react';

type Props = {
  mainText?: string;
  subText?: string;
  size?: 'small' | 'medium';
  props?: {
    mainText?: TypographyProps;
    subText?: TypographyProps;
  };
  dataTestId?: string;
};

const BigNumberMetric: FC<Props> = ({
  mainText,
  subText,
  size = 'medium',
  props,
  dataTestId,
}) => (
  <Box
    style={{ display: 'flex', alignItems: 'baseline' }}
    data-testid={dataTestId}
  >
    {mainText ? (
      <Typography
        variant={size === 'small' ? 'body1' : 'h5'}
        fontWeight="600"
        fontFamily="Roboto Mono, monospace"
        overflow="hidden"
        textOverflow="ellipsis"
        whiteSpace="nowrap"
        {...props?.mainText}
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
        {...props?.subText}
      >
        {subText}
      </Typography>
    )}
  </Box>
);

export default BigNumberMetric;
