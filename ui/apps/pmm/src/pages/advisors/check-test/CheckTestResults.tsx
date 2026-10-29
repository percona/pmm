import { FC } from 'react';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import CloseIcon from '@mui/icons-material/Close';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import Chip from '@mui/material/Chip';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { SxProps, Theme } from '@mui/material/styles';
import { CodeBlock } from '@percona/peak-ui';
import { Messages } from './CheckTest.messages';
import { CheckTestState } from './useCheckTest';

interface CheckTestResultsProps {
  test: CheckTestState;
  sx?: SxProps<Theme>;
}

// outcome panel of the last test run; renders nothing until a run finishes
export const CheckTestResults: FC<CheckTestResultsProps> = ({ test, sx }) => {
  const { results, scriptOutput, error, closeResults } = test;

  if (results === null && error === null) {
    return null;
  }

  return (
    <Stack
      gap={1}
      data-testid="advisor-check-form-test-results"
      // recessed "console" surface so the results read as output,
      // not as another editable field
      sx={[
        {
          p: 1.5,
          pt: 1,
          border: 1,
          borderColor: 'divider',
          borderRadius: 1,
          bgcolor: 'background.default',
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      <Stack direction="row" justifyContent="space-between" alignItems="center">
        <Stack direction="row" alignItems="center" gap={1}>
          <Typography variant="subtitle2">
            {Messages.testResultsTitle}
          </Typography>
          <Chip
            size="small"
            color={error ? 'error' : 'success'}
            icon={error ? <ErrorOutlineIcon /> : <CheckCircleOutlineIcon />}
            label={error ? Messages.testFailure : Messages.testSuccess}
            data-testid="advisor-check-form-test-status"
          />
          {!error && (
            <Typography variant="body2" color="text.secondary">
              {`· ${Messages.testFindings(results?.length ?? 0)}`}
            </Typography>
          )}
        </Stack>
        <IconButton
          size="small"
          aria-label={Messages.closeResults}
          onClick={closeResults}
          data-testid="advisor-check-form-test-results-close"
        >
          <CloseIcon fontSize="small" />
        </IconButton>
      </Stack>
      {error ? (
        <Typography
          variant="body2"
          color="error"
          sx={{ whiteSpace: 'pre-wrap' }}
          data-testid="advisor-check-form-test-error"
        >
          {error}
        </Typography>
      ) : (
        <CodeBlock
          language="json"
          copyable
          content={JSON.stringify(results, null, 2)}
          maxHeight="30vh"
          // an explicit border so the panel stands out from the
          // form fields around it; wrap long lines instead of
          // clipping them behind a horizontal scroll
          sx={{
            overflow: 'auto',
            m: 0,
            border: 1,
            borderColor: 'divider',
            borderRadius: 1,
            whiteSpace: 'pre-wrap',
            overflowWrap: 'anywhere',
          }}
          data-testid="advisor-check-form-test-output"
        />
      )}
      {scriptOutput && (
        <>
          <Typography variant="subtitle2">{Messages.scriptOutput}</Typography>
          <CodeBlock
            copyable
            content={scriptOutput}
            maxHeight="20vh"
            sx={{
              overflow: 'auto',
              m: 0,
              border: 1,
              borderColor: 'divider',
              borderRadius: 1,
              whiteSpace: 'pre-wrap',
              overflowWrap: 'anywhere',
            }}
            data-testid="advisor-check-form-test-script-output"
          />
        </>
      )}
    </Stack>
  );
};
