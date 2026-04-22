import Stack from '@mui/material/Stack';
import { FC, ReactNode } from 'react';
import { MAX_LABEL_WIDTH } from '../../Settings.constants';
import Typography from '@mui/material/Typography';
import Link from '@mui/material/Link';
import { Messages } from '../../Settings.messages';
import ArrowOutwardIcon from '@mui/icons-material/ArrowOutward';

interface Props {
  label: ReactNode;
  description: ReactNode;
  readMoreLink?: string;
  readMoreText?: string;
}

const SettingsFieldLabel: FC<Props> = ({
  label,
  description,
  readMoreLink,
  readMoreText = Messages.tooltipLinkText,
}) => (
  <Stack maxWidth={MAX_LABEL_WIDTH}>
    <Typography variant="h6">{label}</Typography>
    <Typography variant="body2">
      {description}
      {readMoreLink && (
        <>
          {' '}
          <Link href={readMoreLink} target="_blank" rel="noopener noreferrer">
            {readMoreText}
            <ArrowOutwardIcon sx={{ fontSize: 14 }} />
          </Link>
        </>
      )}
    </Typography>
  </Stack>
);

export default SettingsFieldLabel;
