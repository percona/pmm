import type { ReactNode } from 'react';
import { render, screen } from '@testing-library/react';
import ToggleButton from '@mui/material/ToggleButton';
import { FormProvider, useForm } from 'react-hook-form';
import { describe, expect, it } from 'vitest';
import { baseThemeOptions, ThemeContextProvider } from '../../../../design';
import ToggleButtonGroupInputRegular from './toggle-button-group-regular';

const Wrapper = ({ children }: { children: ReactNode }) => {
  const methods = useForm({ defaultValues: { mode: 'a' } });
  return (
    <ThemeContextProvider themeOptions={baseThemeOptions}>
      <FormProvider {...methods}>{children}</FormProvider>
    </ThemeContextProvider>
  );
};

describe('ToggleButtonGroupInputRegular', () => {
  it('rounds the outer corners of the end buttons', () => {
    render(
      <Wrapper>
        <ToggleButtonGroupInputRegular name="mode">
          <ToggleButton value="a">A</ToggleButton>
          <ToggleButton value="b">B</ToggleButton>
          <ToggleButton value="c">C</ToggleButton>
        </ToggleButtonGroupInputRegular>
      </Wrapper>
    );

    expect(screen.getByRole('button', { name: 'A' })).toHaveStyle({
      borderTopLeftRadius: '999px',
    });
    expect(screen.getByRole('button', { name: 'C' })).toHaveStyle({
      borderTopRightRadius: '999px',
    });
  });
});
