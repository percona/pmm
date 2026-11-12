import { TextField, TextFieldProps } from '@mui/material';
import { kebabize } from '@/utils';
import { Controller, FieldValues, useFormContext } from 'react-hook-form';
import { TextInputProps } from './text.types';

type InputChangeEvent = React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>;
type SelectSlotProps = NonNullable<TextFieldProps['slotProps']>['select'];
type SlotPropsWithChange =
  | { onChange?: (event: InputChangeEvent, ...rest: unknown[]) => void }
  | undefined;

// A slot's own onChange replaces the TextField's in MUI, so the form write is composed into it.
const withStoreValue = (slot: unknown, storeValue: (event: InputChangeEvent) => void) => {
  const compose = (props: SlotPropsWithChange) => {
    const slotOnChange = props?.onChange;
    if (!slotOnChange) return props;

    return {
      ...props,
      onChange: (event: InputChangeEvent, ...rest: unknown[]) => {
        storeValue(event);
        slotOnChange(event, ...rest);
      },
    };
  };

  return typeof slot === 'function'
    ? (ownerState: unknown) => compose(slot(ownerState))
    : compose(slot as SlotPropsWithChange);
};

const TextInput = <T extends FieldValues = FieldValues>({
  control,
  name,
  label,
  controllerProps,
  textFieldProps = {},
  isRequired,
  formHelperTextProps = {},
}: TextInputProps<T>) => {
  const formContext = useFormContext<T>();
  const contextControl = formContext?.control;

  const { sx: textFieldPropsSx, onChange, slotProps, ...restFieldProps } = textFieldProps;
  const { sx: formHelperTextPropsSx } = formHelperTextProps;

  return (
    <Controller
      name={name}
      control={control ?? contextControl}
      render={({ field: { onChange: fieldOnChange, ...field }, fieldState: { error } }) => {
        const storeValue = (event: InputChangeEvent) => {
          const transformed: unknown = onChange?.(event);
          const isValue = typeof transformed === 'string' || typeof transformed === 'number';
          fieldOnChange(isValue ? transformed : event);
        };

        return (
          <TextField
            label={label}
            {...field}
            size={restFieldProps?.size || 'small'}
            sx={{ mt: 3, ...textFieldPropsSx }}
            {...restFieldProps}
            onChange={storeValue}
            slotProps={{
              ...slotProps,
              input: withStoreValue(slotProps?.input ?? restFieldProps.InputProps, storeValue),
              select: withStoreValue(
                slotProps?.select ?? restFieldProps.SelectProps,
                storeValue
              ) as SelectSlotProps,
            }}
            variant="outlined"
            required={isRequired}
            error={!!error}
            InputLabelProps={{
              shrink: true,
            }}
            inputProps={{
              'data-testid': `text-input-${kebabize(name)}`,
              onWheel: (e) => {
                (e.target as HTMLElement).blur();
              },
              onBlur: (event: React.FocusEvent<HTMLInputElement>) => {
                if (textFieldProps?.onBlur) {
                  const modifiedEvent = {
                    ...event,
                    target: {
                      ...event.target,
                      value: textFieldProps.onBlur(event),
                    },
                  };
                  fieldOnChange(modifiedEvent);
                }
              },
              ...restFieldProps?.inputProps,
            }}
            helperText={error ? error.message : restFieldProps?.helperText}
            FormHelperTextProps={{
              sx: formHelperTextPropsSx,
            }}
          />
        );
      }}
      {...controllerProps}
    />
  );
};

export default TextInput;
