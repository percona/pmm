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

  // MUI only calls renderTags with a non-empty value, but the component is rendered directly
  // by its own tests and by any future caller, and the code this replaced was total.
  if (count === 0) {
    return null;
  }

  if (tagPresentation === 'label') {
    // Only the first name is spelled out, with a count for the rest. Joining every name
    // produced a string wider than the field, which pushed the autocomplete's own input onto
    // a second line and left the clear button floating below the text.
    const [first, ...rest] = value;
    const allNames = value.map((option) => option.label).join(', ');

    return (
      <Tooltip title={rest.length > 0 ? allNames : ''} arrow>
        <Stack
          direction="row"
          alignItems="center"
          gap={0.75}
          pl={1.5}
          // minWidth 0 lets the name shrink instead of forcing the field wider than its
          // container; without it a flex child refuses to go below its content width.
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
          {/* Never allowed to shrink: the count is what tells the reader the label is only
              part of the selection, so it has to survive when the name is truncated. */}
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
