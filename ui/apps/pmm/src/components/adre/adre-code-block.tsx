import Check from '@mui/icons-material/Check';
import ContentCopy from '@mui/icons-material/ContentCopy';
import { Box, IconButton, Tooltip, Typography } from '@mui/material';
import {
  FC,
  ReactNode,
  isValidElement,
  useEffect,
  useRef,
  useState,
} from 'react';

const COPIED_MS = 1500;

/** The text inside rendered markdown children. */
const textOf = (node: ReactNode): string => {
  if (node == null || typeof node === 'boolean') return '';
  if (typeof node === 'string' || typeof node === 'number') return String(node);
  if (Array.isArray(node)) return node.map(textOf).join('');
  if (isValidElement(node)) {
    return textOf((node.props as { children?: ReactNode }).children);
  }
  return '';
};

/** The language of a fenced block, from the language-x class markdown gives its code. */
const languageOf = (node: ReactNode) => {
  if (!isValidElement(node)) return undefined;
  const className = (node.props as { className?: string }).className ?? '';
  return /language-([\w-]+)/.exec(className)?.[1];
};

/** A fenced code block with its language and a copy button. */
export const AdreCodeBlock: FC<{ children?: ReactNode }> = ({ children }) => {
  const [copied, setCopied] = useState(false);
  const timer = useRef<number | undefined>(undefined);
  useEffect(() => () => window.clearTimeout(timer.current), []);
  const language = languageOf(children);
  // The clipboard is missing outside secure contexts.
  const copy = () =>
    Promise.resolve(navigator.clipboard?.writeText(textOf(children))).then(
      () => {
        setCopied(true);
        window.clearTimeout(timer.current);
        timer.current = window.setTimeout(() => setCopied(false), COPIED_MS);
      },
      () => undefined
    );

  return (
    <Box
      sx={{
        my: 1.5,
        maxWidth: '100%',
        minWidth: 0,
        border: 1,
        borderColor: 'rgba(255,255,255,0.08)',
        borderRadius: '10px',
        bgcolor: '#0D0C0F',
        overflow: 'hidden',
      }}
    >
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          minHeight: 36,
          pl: 1.75,
          pr: 0.5,
          borderBottom: 1,
          borderColor: 'rgba(255,255,255,0.06)',
        }}
      >
        <Typography variant="caption" sx={{ color: '#8E8C95' }}>
          {language ?? 'code'}
        </Typography>
        <Tooltip title={copied ? 'Copied' : 'Copy code'}>
          <IconButton
            size="small"
            aria-label={copied ? 'Copied' : 'Copy code'}
            onClick={copy}
            sx={{ color: copied ? 'success.light' : '#A9A7B0' }}
          >
            {copied ? (
              <Check fontSize="inherit" />
            ) : (
              <ContentCopy fontSize="inherit" />
            )}
          </IconButton>
        </Tooltip>
      </Box>
      <Box
        component="pre"
        sx={{
          m: 0,
          py: 1.5,
          px: 1.75,
          overflowX: 'auto',
          color: '#E6E4EA',
          lineHeight: 1.6,
          // Fenced code: undo the inline <code> chrome so the block reads as one preformatted unit.
          '& code': {
            display: 'block',
            p: 0,
            bgcolor: 'transparent',
            borderRadius: 0,
            whiteSpace: 'pre',
            overflowWrap: 'normal',
            fontSize: '0.8125rem',
          },
        }}
      >
        {children}
      </Box>
    </Box>
  );
};
