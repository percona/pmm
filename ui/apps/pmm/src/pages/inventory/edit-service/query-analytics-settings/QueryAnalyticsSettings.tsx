import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { RadioGroup, SwitchInput, TextInput } from '@percona/peak-ui';
import type { FC } from 'react';
import type { ManagedServiceType } from 'types/services.types';
import { PMM_SERVER_POSTGRESQL, QUERY_SOURCES } from '../EditService.constants';
import { Messages } from '../EditService.messages';
import type { EditServiceFormValues } from '../EditService.schema';
import { QuerySource } from '../EditService.types';
import { SectionHeader } from '../section-header';
import { useIsSettingShown } from '../useIsSettingShown';

const { qan } = Messages;

interface Props {
  serviceType: ManagedServiceType;
  serviceName: string;
  isCloudNode: boolean;
}

export const QueryAnalyticsSettings: FC<Props> = ({
  serviceType,
  serviceName,
  isCloudNode,
}) => {
  const isShown = useIsSettingShown(serviceType);
  const isInternal = serviceName === PMM_SERVER_POSTGRESQL;
  const options = [
    { value: QuerySource.none, label: qan.sources.none, disabled: isInternal },
    ...(QUERY_SOURCES[serviceType] ?? []).map(({ value, isCloud }) => ({
      value,
      label: qan.sources[value],
      disabled: isInternal || (isCloudNode && !isCloud),
    })),
  ];
  const restriction =
    (isInternal && qan.internalPostgresql) || (isCloudNode && qan.cloudNode);

  return (
    <Stack gap={2} data-testid="query-analytics-settings">
      <SectionHeader title={qan.title} description={qan.description} />
      <Stack gap={0.5}>
        <RadioGroup<EditServiceFormValues>
          name="querySource"
          label={qan.querySource}
          options={options}
        />
        {restriction && (
          <Typography
            variant="body2"
            color="text.secondary"
            data-testid="query-source-restriction"
          >
            {restriction}
          </Typography>
        )}
      </Stack>
      {(isShown('maxQueryLength') || isShown('maxSlowlogFileSize')) && (
        <Stack direction={{ xs: 'column', md: 'row' }} gap={2}>
          {isShown('maxQueryLength') && (
            <TextInput<EditServiceFormValues>
              name="maxQueryLength"
              label={qan.maxQueryLength}
              textFieldProps={{
                type: 'number',
                fullWidth: true,
                helperText: qan.maxQueryLengthDescription,
              }}
            />
          )}
          {isShown('maxSlowlogFileSize') && (
            <TextInput<EditServiceFormValues>
              name="maxSlowlogFileSize"
              label={qan.maxSlowlogFileSize}
              textFieldProps={{
                type: 'number',
                fullWidth: true,
                helperText: qan.maxSlowlogFileSizeDescription,
              }}
            />
          )}
        </Stack>
      )}
      {(isShown('disableQueryExamples') ||
        isShown('disableCommentsParsing')) && (
        <Stack>
          {isShown('disableQueryExamples') && (
            <SwitchInput<EditServiceFormValues>
              name="disableQueryExamples"
              label={qan.disableQueryExamples}
            />
          )}
          {isShown('disableCommentsParsing') && (
            <SwitchInput<EditServiceFormValues>
              name="disableCommentsParsing"
              label={qan.disableCommentsParsing}
            />
          )}
        </Stack>
      )}
    </Stack>
  );
};
