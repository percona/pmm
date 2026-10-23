import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { TextSelect } from './TextSelect';
import { TextSelectOption } from './TextSelect.types';

const OPTIONS: TextSelectOption<string>[] = [
  {
    label: 'Option #1',
    value: 'one',
  },
  {
    label: 'Option #2',
    value: 'two',
  },
  {
    label: 'Option #3',
    value: 'three',
  },
];

const onChangeMock = vi.fn();

describe('TextSelect', () => {
  it('shows default label', () => {
    render(
      <TextSelect value="one" options={OPTIONS} onChange={onChangeMock} />
    );

    expect(screen.getByTestId('text-select-button')).toHaveTextContent(
      'Select:'
    );
  });

  it('shows correct label', () => {
    render(
      <TextSelect
        label="Custom"
        value="one"
        options={OPTIONS}
        onChange={onChangeMock}
      />
    );

    expect(screen.getByTestId('text-select-button')).toHaveTextContent(
      'Custom:'
    );
  });

  it('shows selected value', () => {
    render(
      <TextSelect value="two" options={OPTIONS} onChange={onChangeMock} />
    );

    expect(screen.getByTestId('text-select-button')).toHaveTextContent(
      'Option #2'
    );
  });

  it('opens menu when clicked', () => {
    render(
      <TextSelect value="one" options={OPTIONS} onChange={onChangeMock} />
    );

    fireEvent.click(screen.getByTestId('text-select-button'));

    expect(screen.queryByTestId('text-select-menu')).toBeDefined();
  });

  it('closes menu when option selected', async () => {
    render(
      <TextSelect value="one" options={OPTIONS} onChange={onChangeMock} />
    );

    fireEvent.click(screen.getByTestId('text-select-button'));

    fireEvent.click(screen.getByTestId('text-select-option-three'));

    await waitFor(() =>
      expect(screen.queryByTestId('text-select-menu')).toBeNull()
    );
  });

  it('selects correct option', async () => {
    render(
      <TextSelect value="one" options={OPTIONS} onChange={onChangeMock} />
    );

    fireEvent.click(screen.getByTestId('text-select-button'));

    fireEvent.click(screen.getByTestId('text-select-option-three'));

    expect(onChangeMock).toHaveBeenCalledWith('three');
  });

  it('disables the button when disabled is true', () => {
    render(
      <TextSelect
        value="one"
        options={OPTIONS}
        onChange={onChangeMock}
        disabled
      />
    );

    expect(screen.getByTestId('text-select-button')).toBeDisabled();
  });

  it('shows disabled value when disabled is true', () => {
    render(
      <TextSelect
        value="one"
        options={OPTIONS}
        onChange={onChangeMock}
        disabled
        disabledValue="OFF"
      />
    );

    expect(screen.getByTestId('text-select-button')).toHaveTextContent('OFF');
  });

  it('shows start icon when startIcon is provided', () => {
    render(
      <TextSelect
        value="one"
        options={OPTIONS}
        onChange={onChangeMock}
        startIcon={<div data-testid="start-icon" />}
      />
    );

    expect(screen.getByTestId('text-select-button')).toContainElement(
      screen.getByTestId('start-icon')
    );
  });

  it('shows a selected label that differs from its menu label', () => {
    render(
      <TextSelect
        label="Auto-refresh"
        value="two"
        selectedLabel="OFF"
        options={OPTIONS}
        onChange={onChangeMock}
      />
    );

    expect(screen.getByTestId('text-select-button')).toHaveTextContent(
      'Auto-refresh: OFF'
    );
  });

  it('says it opens a menu, and whether it is open', () => {
    render(
      <TextSelect value="one" options={OPTIONS} onChange={onChangeMock} />
    );
    const button = screen.getByTestId('text-select-button');

    expect(button).toHaveAttribute('aria-haspopup', 'menu');
    expect(button).not.toHaveAttribute('aria-expanded');

    fireEvent.click(button);

    expect(button).toHaveAttribute('aria-expanded', 'true');
    expect(button).toHaveAttribute(
      'aria-controls',
      screen.getByTestId('text-select-menu').id
    );
  });

  it('can draw its button alone in the Base theme, with a compact menu under its arrow', () => {
    render(
      <TextSelect
        value="one"
        options={OPTIONS}
        onChange={onChangeMock}
        buttonTheme="base"
        compactMenu
      />
    );

    fireEvent.click(screen.getByTestId('text-select-button'));

    const menu = screen.getByTestId('text-select-menu');

    expect(menu.querySelector('.MuiMenu-paper')).toHaveStyle({ minWidth: '0' });
    // The list keeps its padding above the first option and below the last.
    expect(menu.querySelector('.MuiMenu-list')).toHaveStyle({
      paddingTop: '8px',
      paddingBottom: '8px',
    });
  });
});
