// everest
// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { Stack, Typography } from '@mui/material';
import { LabeledContentProps } from './labeled-content.types';

const LabeledContent = ({
  label,
  caption,
  children: verticalStackChildrenSlot,
  isRequired = false,
  verticalStackSx: verticalSx = {
    '.MuiTextField-root': {},
    '.MuiAutocomplete-root': {},
  },
  horizontalStackSx: horizontalStackSx,
  horizontalStackChildrenSlot,
  ...typographyProps
}: LabeledContentProps) => {
  const {
    // @ts-expect-error
    '.MuiTextField-root': textFieldRootSx,
    // @ts-expect-error
    '.MuiAutocomplete-root': autocompleteRootSx,
    ...verticalStackSx
  } = verticalSx;
  return (
    <Stack
      sx={{
        '.MuiTextField-root': {
          mt: 1.5,
          ...textFieldRootSx,
        },
        '.MuiAutocomplete-root': {
          mt: 1.5,
          ...autocompleteRootSx,
        },
        mt: 2,
        ...verticalStackSx,
      }}
    >
      <Stack
        sx={{
          flexDirection: 'row',
          alignItems: 'center',
          marginBottom: 0.5,
          ...horizontalStackSx,
        }}
      >
        <Typography variant="sectionHeading" {...typographyProps}>
          {label}
          {isRequired && <span>*</span>}
        </Typography>
        {horizontalStackChildrenSlot}
      </Stack>
      {caption && <Typography variant="body2">{caption}</Typography>}
      {verticalStackChildrenSlot}
    </Stack>
  );
};

export default LabeledContent;
