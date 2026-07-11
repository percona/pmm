import { FC } from 'react';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { SwitchInput, TextInput } from '@percona/percona-ui';
import {
  ParamDefinition,
  ParamType,
  Template,
} from 'types/alert-templates.types';
import { beautifyUnit } from 'utils/alert-templates.utils';
import { Messages } from '../CreateAlertFromTemplate.messages';

interface Props {
  template: Template | null;
}

const floatRange = (param: ParamDefinition): string => {
  const parts: string[] = [];
  if (param.float?.min !== undefined) {
    parts.push(`min: ${param.float.min}`);
  }
  if (param.float?.max !== undefined) {
    parts.push(`max: ${param.float.max}`);
  }
  return parts.length ? `[${parts.join(', ')}]` : '';
};

const ParamField: FC<{ param: ParamDefinition }> = ({ param }) => {
  const name = `params.${param.name}`;

  if (param.type === ParamType.BOOL) {
    return <SwitchInput name={name} label={param.summary} />;
  }

  if (param.type === ParamType.STRING) {
    return (
      <TextInput
        name={name}
        label={param.summary}
        isRequired
        controllerProps={{ rules: { required: Messages.validation.required } }}
      />
    );
  }

  return (
    <TextInput
      name={name}
      label={Messages.paramDescription(
        param.summary,
        beautifyUnit(param.unit),
        floatRange(param)
      )}
      isRequired
      textFieldProps={{ type: 'number' }}
      controllerProps={{
        rules: {
          required: Messages.validation.required,
          ...(param.float?.min !== undefined && {
            min: {
              value: param.float.min,
              message: Messages.validation.min(param.float.min),
            },
          }),
          ...(param.float?.max !== undefined && {
            max: {
              value: param.float.max,
              message: Messages.validation.max(param.float.max),
            },
          }),
        },
      }}
    />
  );
};

export const ParamsSection: FC<Props> = ({ template }) => {
  if (!template || template.params.length === 0) {
    return null;
  }

  return (
    <Stack gap={2}>
      <Typography variant="h6">{Messages.sections.params}</Typography>
      {template.params.map((param) => (
        <ParamField key={param.name} param={param} />
      ))}
    </Stack>
  );
};

export default ParamsSection;
