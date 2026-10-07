import { fireEvent, render, screen } from '@testing-library/react';
import type { FC, PropsWithChildren } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { TestWrapper } from 'utils/testWrapper';
import { PasswordField, TextField } from '.';

const Form: FC<PropsWithChildren> = ({ children }) => {
  const methods = useForm({
    defaultValues: {
      serviceName: '',
      password: '',
      limit: '',
      customLabels: '',
    },
  });

  return <FormProvider {...methods}>{children}</FormProvider>;
};

const renderFields = () =>
  render(
    <TestWrapper>
      <Form>
        <TextField
          name="serviceName"
          label="Service name"
          tooltip="Service name to use"
        />
        <PasswordField name="password" label="Password" />
        <TextField name="limit" kind="number" label="Limit" />
        <TextField name="customLabels" kind="textarea" label="Custom labels" />
      </Form>
    </TestWrapper>
  );

describe('fields', () => {
  it('renders each kind of text field', () => {
    renderFields();

    expect(screen.getByTestId('text-input-service-name')).toHaveAttribute(
      'type',
      'text'
    );
    expect(screen.getByTestId('text-input-limit')).toHaveAttribute(
      'type',
      'number'
    );
    expect(screen.getByTestId('text-input-custom-labels').tagName).toBe(
      'TEXTAREA'
    );
  });

  it('explains a field in a tooltip', async () => {
    renderFields();

    fireEvent.mouseOver(screen.getAllByTestId('InfoOutlinedIcon')[0]);

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'Service name to use'
    );
  });

  it('masks the password until revealed', () => {
    renderFields();

    expect(screen.getByTestId('text-input-password')).toHaveAttribute(
      'type',
      'password'
    );

    fireEvent.click(screen.getByTestId('password-reveal'));

    expect(screen.getByTestId('text-input-password')).toHaveAttribute(
      'type',
      'text'
    );
  });
});
