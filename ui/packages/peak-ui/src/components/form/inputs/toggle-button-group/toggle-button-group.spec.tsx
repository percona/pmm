import type { ReactNode } from 'react';
import { render, screen } from '@testing-library/react';
import ToggleButton from '@mui/material/ToggleButton';
import { FormProvider, useForm } from 'react-hook-form';
import { describe, expect, it } from 'vitest';
import type { ToggleButtonGroupInputProps } from './toggle-button-group.types';
import ToggleButtonGroupInput from './toggle-button-group';

const Wrapper = ({ children }: { children: ReactNode }) => {
  const methods = useForm({ defaultValues: { mode: 'a' } });
  return <FormProvider {...methods}>{children}</FormProvider>;
};

const renderGroup = (
  toggleButtonGroupProps: ToggleButtonGroupInputProps['toggleButtonGroupProps']
) =>
  render(
    <Wrapper>
      <ToggleButtonGroupInput
        name="mode"
        toggleButtonGroupProps={toggleButtonGroupProps}
      >
        <ToggleButton value="a">A</ToggleButton>
        <ToggleButton value="b">B</ToggleButton>
      </ToggleButtonGroupInput>
    </Wrapper>
  );

describe('ToggleButtonGroupInput', () => {
  it('applies an sx callback', () => {
    renderGroup({ sx: () => ({ marginTop: '7px' }) });

    expect(screen.getByTestId('toggle-button-group-input-mode')).toHaveStyle({
      marginTop: '7px',
    });
  });

  it('applies an sx array', () => {
    renderGroup({ sx: [{ marginTop: '7px' }, { marginBottom: '3px' }] });

    expect(screen.getByTestId('toggle-button-group-input-mode')).toHaveStyle({
      marginTop: '7px',
      marginBottom: '3px',
    });
  });
});
