import { FC } from 'react';
import { Box } from '@mui/material';
import UnavailableText from 'components/unavailable-text';

interface Props {
  description?: string;
}

const AlertDescription: FC<Props> = ({ description }) => {
  if (!description) {
    return <UnavailableText />;
  }

  return <Box sx={{ whiteSpace: 'pre-wrap' }}>{description}</Box>;
};

export default AlertDescription;
