import {
  FormControlLabel,
  RadioGroup as MuiRadioGroup,
  Radio,
} from '@mui/material';
import { Controller, useFormContext } from 'react-hook-form';
import LabeledContent from '../../../labeled-content';
import { RadioGroupProps } from './radio-group.types';

const RadioGroup = ({
  name,
  control,
  label,
  controllerProps,
  labelProps,
  radioGroupFieldProps,
  isRequired = false,
  options,
}: RadioGroupProps) => {
  const { control: contextControl } = useFormContext();
  const content = (
    <Controller
      name={name}
      control={control ?? contextControl}
      render={({ field }) => (
        <MuiRadioGroup
          {...field}
          row
          name={`radio-group-${name}`}
          {...radioGroupFieldProps}
        >
          {options.map((option) => (
            <FormControlLabel
              key={option.label}
              value={option.value}
              control={
                <Radio
                  {...option.radioProps}
                  // @ts-expect-error
                  inputProps={{ 'data-testid': `radio-option-${option.value}` }}
                />
              }
              label={option.label}
              disabled={option.disabled}
            />
          ))}
        </MuiRadioGroup>
      )}
      {...controllerProps}
    />
  );

  return label ? (
    <LabeledContent label={label} isRequired={isRequired} {...labelProps}>
      {content}
    </LabeledContent>
  ) : (
    content
  );
};

export default RadioGroup;
