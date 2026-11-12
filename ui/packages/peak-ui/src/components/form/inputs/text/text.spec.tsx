import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { FormProvider, useForm, useFormContext, useWatch } from 'react-hook-form';
import { MenuItem, TextFieldProps } from '@mui/material';
import TextInput from './text';

const StoredValue = () => {
  const value = useWatch({ name: 'host' });
  return <span data-testid="stored-value">{JSON.stringify(value)}</span>;
};

const Form = ({
  textFieldProps,
  children,
}: {
  textFieldProps?: TextFieldProps;
  children?: React.ReactNode;
}) => {
  const methods = useForm({ defaultValues: { host: '' } });

  return (
    <FormProvider {...methods}>
      {children ?? <TextInput name="host" textFieldProps={textFieldProps} />}
      <StoredValue />
    </FormProvider>
  );
};

const type = (value: string) =>
  fireEvent.change(screen.getByTestId('text-input-host'), { target: { value } });

const storedValue = () => JSON.parse(screen.getByTestId('stored-value').textContent ?? '');

describe('TextInput', () => {
  it('stores the raw text when no onChange is given', () => {
    render(<Form />);

    type('abc');

    expect(storedValue()).toBe('abc');
    expect(screen.getByTestId('text-input-host')).toHaveValue('abc');
  });

  it('stores the value returned by textFieldProps.onChange', () => {
    const toUpper = (e: React.ChangeEvent<HTMLInputElement>) => e.target.value.toUpperCase();
    render(<Form textFieldProps={{ onChange: toUpper as TextFieldProps['onChange'] }} />);

    type('abc');

    expect(storedValue()).toBe('ABC');
    expect(screen.getByTestId('text-input-host')).toHaveValue('ABC');
  });

  it('stores the raw text when textFieldProps.onChange returns nothing', () => {
    const onChange = vi.fn();
    render(<Form textFieldProps={{ onChange }} />);

    type('abc');

    expect(onChange).toHaveBeenCalledTimes(1);
    expect(storedValue()).toBe('abc');
  });

  it('stores the raw text when textFieldProps.onChange returns a promise', () => {
    const onChange = vi.fn(async () => undefined);
    render(<Form textFieldProps={{ onChange }} />);

    type('abc');

    expect(storedValue()).toBe('abc');
  });

  it('stores the text when InputProps.onChange is given, and calls it after the store', () => {
    let storedWhenCalled: unknown;
    const Probe = () => {
      const { getValues } = useFormContext();
      return (
        <TextInput
          name="host"
          textFieldProps={{
            InputProps: { onChange: () => (storedWhenCalled = getValues('host')) },
          }}
        />
      );
    };
    render(
      <Form>
        <Probe />
      </Form>
    );

    type('abc');

    expect(storedValue()).toBe('abc');
    expect(storedWhenCalled).toBe('abc');
  });

  it('stores the text when slotProps.input is a callback with its own onChange', () => {
    const onInputChange = vi.fn();
    render(<Form textFieldProps={{ slotProps: { input: () => ({ onChange: onInputChange }) } }} />);

    type('abc');

    expect(onInputChange).toHaveBeenCalledTimes(1);
    expect(storedValue()).toBe('abc');
  });

  it('stores the chosen option of a native select with its own SelectProps.onChange', () => {
    const onSelectChange = vi.fn();
    render(
      <Form
        textFieldProps={{
          select: true,
          SelectProps: { native: true, onChange: onSelectChange },
          children: [
            <option key="" value="" />,
            <option key="b" value="b">
              B
            </option>,
          ],
        }}
      />
    );

    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'b' } });

    expect(onSelectChange).toHaveBeenCalledTimes(1);
    expect(storedValue()).toBe('b');
  });

  it('stores the chosen option of a select with its own SelectProps.onChange', () => {
    const onSelectChange = vi.fn();
    render(
      <Form
        textFieldProps={{
          select: true,
          SelectProps: { onChange: onSelectChange },
          children: [
            <MenuItem key="a" value="a">
              A
            </MenuItem>,
            <MenuItem key="b" value="b">
              B
            </MenuItem>,
          ],
        }}
      />
    );

    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.click(screen.getByRole('option', { name: 'B' }));

    expect(onSelectChange).toHaveBeenCalledTimes(1);
    expect(storedValue()).toBe('b');
  });

  it('stores the chosen option of a select and calls its InputProps.onChange', () => {
    const onInputChange = vi.fn();
    render(
      <Form
        textFieldProps={{
          select: true,
          InputProps: { onChange: onInputChange },
          children: [
            <MenuItem key="a" value="a">
              A
            </MenuItem>,
            <MenuItem key="b" value="b">
              B
            </MenuItem>,
          ],
        }}
      />
    );

    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.click(screen.getByRole('option', { name: 'B' }));

    expect(onInputChange).toHaveBeenCalledTimes(1);
    expect(storedValue()).toBe('b');
  });

  it('stores the value returned by textFieldProps.onBlur', () => {
    const trim = (e: React.FocusEvent<HTMLInputElement>) => e.target.value.trim();
    render(<Form textFieldProps={{ onBlur: trim as TextFieldProps['onBlur'] }} />);

    type('  abc  ');
    fireEvent.blur(screen.getByTestId('text-input-host'));

    expect(storedValue()).toBe('abc');
  });
});
