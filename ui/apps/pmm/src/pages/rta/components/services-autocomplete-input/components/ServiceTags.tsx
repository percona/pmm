import { AutocompleteRenderGetTagProps } from '@mui/material/Autocomplete';
import {
  ServiceOption,
  TagPresentation,
} from '../ServicesAutocompleteInput.types';
import { FC } from 'react';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import Chip from '@mui/material/Chip';

interface Props {
  tagPresentation: TagPresentation;
  value: ServiceOption[];
  getTagProps: AutocompleteRenderGetTagProps;
}

const ServiceTags: FC<Props> = ({ tagPresentation, value, getTagProps }) => {
  const count = value.length;

  if (tagPresentation === 'label') {
    return (
      <Stack pl={1.5}>
        <Typography
          variant="inputText"
          sx={{
            whiteSpace: 'nowrap',
            maxWidth: 360,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            display: 'block',
          }}
        >
          {value.map((option) => option.label).join(', ')}
        </Typography>
      </Stack>
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
