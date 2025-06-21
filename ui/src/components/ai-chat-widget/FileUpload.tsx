import React, { useRef, useState } from 'react';
import {
  Box,
  IconButton,
  Chip,
  Typography,
  Tooltip,
  Alert,
} from '@mui/material';
import {
  AttachFile as AttachFileIcon,
  Close as CloseIcon,
} from '@mui/icons-material';
import { FileAttachment } from '../../api/aichat';

interface FileUploadProps {
  attachments: FileAttachment[];
  onAttachmentsChange: (attachments: FileAttachment[]) => void;
  maxFileSize?: number; // in bytes
  maxFiles?: number;
  acceptedTypes?: string[]; // MIME types
  triggerRef?: React.MutableRefObject<(() => void) | null>;
}

interface FileUploadButtonProps {
  onClick: () => void;
  disabled?: boolean;
  hasAttachments?: boolean;
  maxFiles: number;
  maxFileSize: number;
}

export const FileUpload: React.FC<FileUploadProps> = ({
  attachments,
  onAttachmentsChange,
  maxFileSize = 10 * 1024 * 1024, // 10MB default (matches backend limit)
  maxFiles = 5,
  acceptedTypes = [
    'text/plain',
    'text/html',
    'text/css',
    'text/javascript',
    'text/markdown',
    'application/json',
    'application/xml',
    'text/xml',
    'application/yaml',
    'text/yaml',
    'image/png',
    'image/jpeg',
    'image/gif',
    'image/webp',
    'application/pdf',
  ],
  triggerRef,
}) => {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [error, setError] = useState<string | null>(null);

  const handleFileSelect = () => {
    fileInputRef.current?.click();
  };

  // Expose file selection method to parent
  React.useEffect(() => {
    if (triggerRef) {
      triggerRef.current = handleFileSelect;
    }
  }, [triggerRef]);

  const handleFileChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files || []);
    setError(null);

    if (attachments.length + files.length > maxFiles) {
      setError(`Maximum ${maxFiles} files allowed`);
      return;
    }

    const newAttachments: FileAttachment[] = [];

    for (const file of files) {
      // Check file size
      if (file.size > maxFileSize) {
        setError(`File "${file.name}" is too large. Maximum size is ${formatFileSize(maxFileSize)}`);
        continue;
      }

      // Check file type
      if (acceptedTypes.length > 0 && !acceptedTypes.includes(file.type)) {
        setError(`File type "${file.type}" is not supported`);
        continue;
      }

      try {
        const base64Content = await fileToBase64(file);
        newAttachments.push({
          filename: file.name,
          content: base64Content,
          mime_type: file.type,
          size: file.size,
        });
      } catch (error) {
        console.error('Error reading file:', error);
        setError(`Failed to read file "${file.name}"`);
      }
    }

    if (newAttachments.length > 0) {
      onAttachmentsChange([...attachments, ...newAttachments]);
    }

    // Clear the input
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleRemoveAttachment = (index: number) => {
    const newAttachments = attachments.filter((_, i) => i !== index);
    onAttachmentsChange(newAttachments);
    setError(null);
  };

  const fileToBase64 = (file: File): Promise<string> => {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.readAsDataURL(file);
      reader.onload = () => {
        if (typeof reader.result === 'string') {
          // Remove the data URL prefix (e.g., "data:text/plain;base64,")
          const base64 = reader.result.split(',')[1];
          resolve(base64);
        } else {
          reject(new Error('Failed to read file as base64'));
        }
      };
      reader.onerror = reject;
    });
  };

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getFileIcon = (mimeType: string): string => {
    if (mimeType.startsWith('image/')) return '🖼️';
    if (mimeType.startsWith('text/')) return '📄';
    if (mimeType.includes('json')) return '📋';
    if (mimeType.includes('pdf')) return '📕';
    return '📎';
  };

  return (
    <>
      {/* File input (hidden) */}
      <input
        ref={fileInputRef}
        type="file"
        multiple
        accept={acceptedTypes.join(',')}
        onChange={handleFileChange}
        style={{ display: 'none' }}
      />

      {/* Error message */}
      {error && (
        <Alert severity="error" sx={{ mb: 1 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* Attached files */}
      {attachments.length > 0 && (
        <Box sx={{ mb: 1 }}>
          <Typography variant="caption" color="textSecondary" sx={{ mb: 0.5, display: 'block' }}>
            Attached files ({attachments.length}):
          </Typography>
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
            {attachments.map((attachment, index) => (
              <Chip
                key={index}
                size="small"
                label={
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                    <span>{getFileIcon(attachment.mime_type)}</span>
                    <span>{attachment.filename}</span>
                    <Typography variant="caption" color="textSecondary">
                      ({formatFileSize(attachment.size)})
                    </Typography>
                  </Box>
                }
                onDelete={() => handleRemoveAttachment(index)}
                deleteIcon={<CloseIcon />}
                variant="outlined"
                sx={{ maxWidth: '100%' }}
              />
            ))}
          </Box>
        </Box>
      )}
    </>
  );
};

// Separate button component for use in input area
export const FileUploadButton: React.FC<FileUploadButtonProps> = ({
  onClick,
  disabled = false,
  hasAttachments = false,
  maxFiles,
  maxFileSize,
}) => {
  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  return (
    <Tooltip title={`Attach files (max ${maxFiles}, ${formatFileSize(maxFileSize)} each)`}>
      <IconButton
        size="small"
        onClick={onClick}
        disabled={disabled}
        sx={{ 
          alignSelf: 'flex-end',
          color: hasAttachments ? 'primary.main' : 'text.secondary'
        }}
      >
        <AttachFileIcon />
      </IconButton>
    </Tooltip>
  );
}; 