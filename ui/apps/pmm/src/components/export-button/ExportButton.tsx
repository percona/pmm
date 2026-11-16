import FileDownloadOutlined from '@mui/icons-material/FileDownloadOutlined';
import Button, { type ButtonProps } from '@mui/material/Button';
import type { FC } from 'react';
import { Messages } from './ExportButton.messages';

// Shared by the pages that export their table to a file
export const ExportButton: FC<ButtonProps> = (props) => (
  <Button startIcon={<FileDownloadOutlined />} {...props}>
    {Messages.export}
  </Button>
);
