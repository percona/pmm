import { AutocompleteRenderGetTagProps } from '@mui/material/Autocomplete';
import {
  ServiceOption,
  TagPresentation,
} from '../ServicesAutocompleteInput.types';
import { FC } from 'react';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import Chip from '@mui/material/Chip';
import Tooltip from '@mui/material/Tooltip';
import { Messages } from '../ServicesAutocompleteInput.messages';

interface Props {
  tagPresentation: TagPresentation;
  value: ServiceOption[];
  getTagProps: AutocompleteRenderGetTagProps;
}

const ServiceTags: FC<Props> = ({ tagPresentation, value, getTagProps }) => {
  const count = value.length;

  // MUI never calls renderTags empty, but the component is also rendered directly.
  if (count === 0) {
    return null;
  }

  if (tagPresentation === 'label') {
    // One name plus a count: every name would push the input and clear button to a second line.
    const [first, ...rest] = value;
    const allNames = value.map((option) => option.label).join(', ');

    return (
      <Tooltip title={rest.length > 0 ? allNames : ''} arrow>
        <Stack
          direction="row"
          alignItems="center"
          gap={0.75}
          pl={1.5}
          // minWidth 0 lets the name shrink below its content width rather than widen the field.
          sx={{ minWidth: 0, maxWidth: '100%', overflow: 'hidden' }}
        >
          <Typography
            variant="inputText"
            sx={{
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              minWidth: 0,
            }}
          >
            {first.label}
          </Typography>
          {/* Never shrinks: the count shows the label is only part of the selection. */}
          {rest.length > 0 && (
            <Typography
              variant="inputText"
              color="text.secondary"
              sx={{ flexShrink: 0, whiteSpace: 'nowrap' }}
            >
              {Messages.moreServices(rest.length)}
            </Typography>
          )}
        </Stack>
      </Tooltip>
    );
  }

  return (
    <Stack
      direction="row"
      gap={0.5}
      py={0.5}
      alignItems="center"
      flexWrap="wrap"
    >
      {value.slice(0, 2).map((option, index) => (
        <Chip
          size="small"
          label={option.label}
          {...getTagProps({ index })}
          key={option.label}
        />
      ))}
      {value.length > 2 && (
        <Typography variant="inputText">+{count - 2}</Typography>
      )}
    </Stack>
  );
};

export default ServiceTags;
