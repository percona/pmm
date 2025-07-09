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
});
