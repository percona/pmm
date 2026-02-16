import Typography from "@mui/material/Typography";
import { FC } from "react";

type Props = {
  mainText?: string;
  subText?: string;
  tertiaryText?: string;
  size?: 'small' | 'medium';
}

const BigNumberMetric: FC<Props> = ({ mainText, subText, tertiaryText, size = 'medium' }) => (
  <span style={{ display: 'inline-flex', alignItems: 'baseline' }}>
    <Typography variant={size === 'small' ? 'body1' : 'h5'} fontWeight="600" fontFamily='Roboto Mono, monospace'>{mainText || 'N/A'}</Typography>
    {subText && <Typography variant="body2" fontWeight="400" fontFamily='Roboto Mono, monospace' ml={0.5}>{subText}</Typography>}
    {tertiaryText && <Typography variant="body2" color='text.disabled' fontWeight="400" fontFamily='Roboto Mono, monospace' ml={2}>{tertiaryText}</Typography>}
  </span>
)

export default BigNumberMetric;