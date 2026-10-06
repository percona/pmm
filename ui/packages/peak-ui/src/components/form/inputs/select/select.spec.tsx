import type { ReactNode } from 'react';
import { render, screen } from '@testing-library/react';
import MenuItem from '@mui/material/MenuItem';
import { FormProvider, useForm } from 'react-hook-form';
import SelectInput from './select';

const Wrapper = ({ children }: { children: ReactNode }) => {
  const methods = useForm({ defaultValues: { region: '' } });
  return <FormProvider {...methods}>{children}</FormProvider>;
};

const renderSelect = (isRequired?: boolean) =>
  render(
    <Wrapper>
      <SelectInput name="region" label="Region" isRequired={isRequired}>
        <MenuItem value="eu">EU</MenuItem>
      </SelectInput>
    </Wrapper>
  );

describe('SelectInput isRequired', () => {
  it('adds the asterisk to the label and marks the field required', () => {
    renderSelect(true);
    expect(screen.getByText('*')).toBeInTheDocument();
    expect(screen.getByTestId('select-input-region')).toBeRequired();
  });

  it('shows no required marker by default', () => {
    renderSelect();
    expect(screen.queryByText('*')).not.toBeInTheDocument();
    expect(screen.getByTestId('select-input-region')).not.toBeRequired();
  });
});
