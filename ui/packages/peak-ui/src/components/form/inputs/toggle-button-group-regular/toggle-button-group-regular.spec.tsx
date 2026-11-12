import type { ReactNode } from 'react';
import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { createTheme, ThemeProvider, type Theme } from '@mui/material/styles';
import type { ToggleButtonGroupProps } from '@mui/material/ToggleButtonGroup';
import { FormProvider, useForm } from 'react-hook-form';
import { getThemeOptions } from '../../../../design';
import ToggleRegularButton from '../../../buttons/toggle-regular-button';
import ToggleButtonGroupInput from '../toggle-button-group/toggle-button-group';
import ToggleButtonGroupInputRegular from './toggle-button-group-regular';

const theme = createTheme(getThemeOptions('base')('light'));
const PILL = `${theme.shape.borderRadiusFull}px`;

const Wrapper = ({ children, appTheme = theme }: { children: ReactNode; appTheme?: Theme }) => {
  const methods = useForm({ defaultValues: { view: 'table' } });
  return (
    <ThemeProvider theme={appTheme}>
      <FormProvider {...methods}>{children}</FormProvider>
    </ThemeProvider>
  );
};

const options = [
  <ToggleRegularButton key="table" value="table" dataTestId="btn-table">
    Table
  </ToggleRegularButton>,
  <ToggleRegularButton key="graph" value="graph" dataTestId="btn-graph">
    Graph
  </ToggleRegularButton>,
  <ToggleRegularButton key="raw" value="raw" dataTestId="btn-raw">
    Raw
  </ToggleRegularButton>,
];

const renderRegular = (toggleButtonGroupProps?: ToggleButtonGroupProps, appTheme?: Theme) =>
  render(
    <Wrapper appTheme={appTheme}>
      <ToggleButtonGroupInputRegular name="view" toggleButtonGroupProps={toggleButtonGroupProps}>
        {options}
      </ToggleButtonGroupInputRegular>
    </Wrapper>
  );

const group = () => screen.getByTestId('toggle-button-group-input-view');

// jsdom's getComputedStyle ignores specificity, so read the declarations of the rules that match.
const matches = (element: Element, selector: string) => {
  try {
    return element.matches(selector);
  } catch {
    return false; // jsdom rejects vendor pseudo-elements such as ::-moz-focus-inner
  }
};

const matchingRules = (testId: string) => {
  const element = screen.getByTestId(testId);
  return Array.from(document.styleSheets)
    .flatMap((sheet) => Array.from(sheet.cssRules))
    .filter(
      (rule): rule is CSSStyleRule =>
        rule instanceof CSSStyleRule && matches(element, rule.selectorText)
    );
};

const declared = (testId: string, property: string, selectorPart = '') =>
  matchingRules(testId)
    .filter((rule) => rule.selectorText.includes(selectorPart))
    .map((rule) => rule.style.getPropertyValue(property))
    .filter(Boolean);

describe('ToggleButtonGroupInputRegular', () => {
  it('renders as a fit-content pill instead of full width', () => {
    renderRegular();
    expect(group()).not.toHaveClass('MuiToggleButtonGroup-fullWidth');
    expect(group()).toHaveStyle({ width: 'fit-content' });
    expect(screen.getByTestId('btn-table')).not.toHaveClass('MuiToggleButton-fullWidth');
  });

  it('sizes each button to its label instead of splitting the width evenly', () => {
    renderRegular();
    expect(declared('btn-raw', 'flex-shrink')).toContain('0');
    expect(declared('btn-raw', 'flex-basis')).toContain('auto');
  });

  it('rounds only the outer corners', () => {
    renderRegular();
    expect(declared('btn-table', 'border-radius')).toContain(`${PILL} 0 0 ${PILL}`);
    expect(declared('btn-graph', 'border-radius')).toContain('0px');
    expect(declared('btn-raw', 'border-radius')).toContain(`0 ${PILL} ${PILL} 0`);
  });

  it('leaves no gap or border between buttons', () => {
    renderRegular();
    ['btn-graph', 'btn-raw'].forEach((id) => {
      expect(declared(id, 'margin-left')).toContain('0px');
      expect(declared(id, 'border-left-width')).toContain('0px');
    });
    ['btn-table', 'btn-graph'].forEach((id) =>
      expect(declared(id, 'border-right-width')).toContain('0px')
    );
  });

  it('keeps the pill styles when the consumer passes its own sx', () => {
    renderRegular({ sx: (t) => ({ backgroundColor: t.palette.primary.main }) });
    expect(group()).toHaveStyle({
      width: 'fit-content',
      backgroundColor: theme.palette.primary.main,
    });
  });

  it('lets the consumer opt back into full width', () => {
    renderRegular({ fullWidth: true });
    expect(group()).toHaveClass('MuiToggleButtonGroup-fullWidth');
    expect(group()).toHaveStyle({ width: '100%' });
  });

  it('closes the gap between disabled buttons', () => {
    renderRegular({ disabled: true });
    ['btn-graph', 'btn-raw'].forEach((id) =>
      expect(declared(id, 'border-left-width', 'Mui-disabled')).toContain('0px')
    );
    ['btn-table', 'btn-graph'].forEach((id) =>
      expect(declared(id, 'border-right-width', 'Mui-disabled')).toContain('0px')
    );
  });

  it('stays a horizontal pill where the PMM theme stacks groups on small screens', () => {
    renderRegular(undefined, createTheme(getThemeOptions('pmm')('light')));
    const smallScreenDirections = Array.from(document.styleSheets)
      .flatMap((sheet) => Array.from(sheet.cssRules))
      .filter(
        (rule): rule is CSSMediaRule =>
          rule instanceof CSSMediaRule && rule.conditionText.includes('max-width')
      )
      .flatMap((rule) => Array.from(rule.cssRules))
      .filter(
        (rule): rule is CSSStyleRule =>
          rule instanceof CSSStyleRule && matches(group(), rule.selectorText)
      )
      .map((rule) => rule.style.getPropertyValue('flex-direction'))
      .filter(Boolean);
    expect(smallScreenDirections).toEqual(['column', 'row']);
  });

  it('keeps the stacked MUI corners on a vertical group', () => {
    renderRegular({ orientation: 'vertical' });
    expect(declared('btn-table', 'border-radius')).not.toContain(`${PILL} 0 0 ${PILL}`);
    expect(declared('btn-raw', 'border-radius')).not.toContain(`0 ${PILL} ${PILL} 0`);
    expect(group()).not.toHaveStyle({ width: 'fit-content' });
  });
});

describe('ToggleButtonGroupInput', () => {
  it('stays full width by default', () => {
    render(
      <Wrapper>
        <ToggleButtonGroupInput name="view">{options}</ToggleButtonGroupInput>
      </Wrapper>
    );
    expect(group()).toHaveClass('MuiToggleButtonGroup-fullWidth');
  });

  it('applies an sx callback instead of dropping it', () => {
    render(
      <Wrapper>
        <ToggleButtonGroupInput
          name="view"
          toggleButtonGroupProps={{ sx: () => ({ width: '123px' }) }}
        >
          {options}
        </ToggleButtonGroupInput>
      </Wrapper>
    );
    expect(group()).toHaveStyle({ width: '123px' });
  });
});
