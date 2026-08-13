import React from 'react';
import { act, fireEvent, render } from '@testing-library/react';
import { Search } from './Search';

describe('Search::', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  it('renders correctly', () => {
    const { container } = render(<Search handleSearch={() => {}} />);
    const form = container.querySelector('form');

    expect(form?.children.length).toBe(2);
  });

  it('renders correctly with initial value', () => {
    const { container } = render(
      <Search handleSearch={() => {}} initialValue="Test value" />,
    );

    expect(container.querySelector('input')?.value).toEqual('Test value');
  });

  it('submits correctly', () => {
    const handleSearch = jest.fn();
    const { container } = render(<Search handleSearch={handleSearch} />);

    const form = container.querySelector('form');

    if (form) {
      fireEvent.submit(form);
    }

    expect(handleSearch).toHaveBeenCalled();
  });

  it('searches after debounce when the input changes', () => {
    const handleSearch = jest.fn();
    const { container } = render(<Search handleSearch={handleSearch} />);
    const input = container.querySelector('input');

    fireEvent.change(input!, { target: { value: 'select' } });

    expect(handleSearch).not.toHaveBeenCalled();

    act(() => {
      jest.advanceTimersByTime(300);
    });

    expect(handleSearch).toHaveBeenCalledWith({ search: 'select' });
  });

  it('clears search when the input is emptied', () => {
    const handleSearch = jest.fn();
    const { container } = render(
      <Search handleSearch={handleSearch} initialValue="select" />,
    );
    const input = container.querySelector('input');

    fireEvent.change(input!, { target: { value: '' } });

    act(() => {
      jest.advanceTimersByTime(300);
    });

    expect(handleSearch).toHaveBeenCalledWith({ search: '' });
  });

  it('submits immediately and cancels pending debounced search', () => {
    const handleSearch = jest.fn();
    const { container } = render(<Search handleSearch={handleSearch} />);
    const input = container.querySelector('input');
    const form = container.querySelector('form');

    fireEvent.change(input!, { target: { value: 'pending' } });
    fireEvent.submit(form!);

    expect(handleSearch).toHaveBeenCalledTimes(1);
    expect(handleSearch).toHaveBeenCalledWith({ search: 'pending' });

    act(() => {
      jest.advanceTimersByTime(300);
    });

    expect(handleSearch).toHaveBeenCalledTimes(1);
  });
});
