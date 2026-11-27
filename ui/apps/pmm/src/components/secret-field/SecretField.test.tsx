import { fireEvent, render, screen } from '@testing-library/react';
import type { FC, PropsWithChildren } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { TestWrapper } from 'utils/testWrapper';
import { SecretField } from './SecretField';

const Form: FC<PropsWithChildren> = ({ children }) => {
  const methods = useForm({ defaultValues: { secret: 'abc' } });

  return <FormProvider {...methods}>{children}</FormProvider>;
};

const renderField = (helperText?: string) =>
  render(
    <TestWrapper>
      <Form>
        <SecretField
          name="secret"
          label="Secret"
          testId="secret-input"
          helperText={helperText}
        />
      </Form>
    </TestWrapper>
  );

describe('SecretField', () => {
  it('starts masked and reveals on toggle', () => {
    renderField('Pasted from the ticket.');

    expect(screen.getByTestId('secret-input')).toHaveAttribute(
      'type',
      'password'
    );

    fireEvent.click(screen.getByTestId('secret-input-reveal'));

    expect(screen.getByTestId('secret-input')).toHaveAttribute('type', 'text');
    expect(screen.getByTestId('secret-input-reveal')).toHaveAccessibleName(
      'Hide value'
    );
  });

  it('renders without helper text', () => {
    renderField();

    expect(screen.getByTestId('secret-input')).toHaveValue('abc');
  });
});
