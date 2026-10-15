import { FC } from 'react';
import { Box, Stack } from '@mui/material';
import UnavailableText from 'components/unavailable-text';

// Leading indentation plus a list marker such as "1.", "2)", "-", "*" or "•".
const LIST_ITEM_REGEX = /^(\s*(?:\d+[.)]|[-*•])\s+)(.*)$/;

interface Props {
  description?: string;
}

const AlertDescription: FC<Props> = ({ description }) => {
  if (!description) {
    return <UnavailableText />;
  }

  return (
    <Box>
      {description.split('\n').map((line, index) => {
        const listItem = line.match(LIST_ITEM_REGEX);

        if (listItem) {
          // The marker gets its own column so wrapped text lines up after it.
          return (
            <Stack key={index} direction="row">
              <Box component="span" sx={{ whiteSpace: 'pre', flexShrink: 0 }}>
                {listItem[1]}
              </Box>
              <Box component="span" sx={{ whiteSpace: 'pre-wrap' }}>
                {listItem[2]}
              </Box>
            </Stack>
          );
        }

        return (
          <Box key={index} sx={{ whiteSpace: 'pre-wrap' }}>
            {line || '\u00a0'}
          </Box>
        );
      })}
    </Box>
  );
};

export default AlertDescription;
