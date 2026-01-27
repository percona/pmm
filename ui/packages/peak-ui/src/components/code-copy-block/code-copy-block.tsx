
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

import { Alert } from '@mui/material';
import { CodeCopyBlockProps } from './code-copy-block.types';
import CopyToClipboardButton from '../buttons/copy-to-clipboard-button';

const CodeCopyBlock = ({ message, showCopyButtonText }: CodeCopyBlockProps) => {
  return (
    <Alert
      severity="info"
      icon={false}
      sx={{
        mt: 0.5,
        mb: 0.5,
        '& .MuiAlert-action': {
          alignItems: 'center',
          pt: 0,
        },
        fontFamily: '"Roboto Mono", "Helvetica", "Arial", "sans-serif"',
      }}
      action={
        <CopyToClipboardButton
          showCopyButtonText={showCopyButtonText}
          buttonProps={{ size: 'small', color: 'primary' }}
          textToCopy={message}
        />
      }
    >
      {message}
    </Alert>
  );
};

export default CodeCopyBlock;
