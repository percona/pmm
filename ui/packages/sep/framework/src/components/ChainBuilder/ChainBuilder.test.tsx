/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ChainBuilder, type ChainValue } from './ChainBuilder';

const TASKS = [
  { name: 'task-a' },
  { name: 'task-b' },
  { name: 'task-c' },
  { name: 'task-self' },
];

interface HarnessProps {
  initial?: ChainValue;
  currentTaskName?: string;
  availableTasks?: { name: string }[];
  onChangeSpy?: (v: ChainValue) => void;
  disabled?: boolean;
}

function Harness({
  initial = { chain_task_names: [], chain_on_failure: false },
  currentTaskName = 'task-self',
  availableTasks = TASKS,
  onChangeSpy,
  disabled,
}: HarnessProps) {
  const [value, setValue] = useState<ChainValue>(initial);
  return (
    <ChainBuilder
      availableTasks={availableTasks}
      currentTaskName={currentTaskName}
      value={value}
      onChange={(next) => {
        onChangeSpy?.(next);
        setValue(next);
      }}
      disabled={disabled}
    />
  );
}

async function pickTaskFromDropdown(taskName: string) {
  const user = userEvent.setup();
  const combobox = screen.getByRole('combobox');
  await user.click(combobox);
  const option = await screen.findByRole('option', { name: taskName });
  await user.click(option);
}

describe('ChainBuilder rendering', () => {
  it('renders no chip sequence and no failure toggle when chain is empty', () => {
    render(<Harness />);
    expect(screen.queryByTestId('chain-sequence')).not.toBeInTheDocument();
    expect(
      screen.queryByTestId('chain-on-failure-checkbox')
    ).not.toBeInTheDocument();
    expect(screen.getByRole('combobox')).toBeInTheDocument();
  });

  it('exposes the widget as a labelled group', () => {
    render(<Harness />);
    const group = screen.getByRole('group', {
      name: /chain tasks after execution/i,
    });
    expect(group).toBeInTheDocument();
  });

  it('renders chips and failure toggle when chain is non-empty', () => {
    render(
      <Harness
        initial={{
          chain_task_names: ['task-a', 'task-b'],
          chain_on_failure: false,
        }}
      />
    );
    const sequence = screen.getByTestId('chain-sequence');
    expect(within(sequence).getByText('task-a')).toBeInTheDocument();
    expect(within(sequence).getByText('task-b')).toBeInTheDocument();
    expect(screen.getByTestId('chain-on-failure-checkbox')).toBeInTheDocument();
  });
});

describe('ChainBuilder add', () => {
  it('appends the selected task and emits the new chain', async () => {
    const onChange = vi.fn();
    render(<Harness onChangeSpy={onChange} />);

    await pickTaskFromDropdown('task-a');

    expect(onChange).toHaveBeenCalledWith({
      chain_task_names: ['task-a'],
      chain_on_failure: false,
    });
    expect(
      within(screen.getByTestId('chain-sequence')).getByText('task-a')
    ).toBeInTheDocument();
  });

  it('disables the current task in the dropdown (cycle prevention)', async () => {
    const user = userEvent.setup();
    render(<Harness />);
    await user.click(screen.getByRole('combobox'));
    const option = await screen.findByRole('option', { name: 'task-self' });
    expect(option).toHaveAttribute('aria-disabled', 'true');
  });

  it('disables already-chained tasks in the dropdown', async () => {
    const user = userEvent.setup();
    render(
      <Harness
        initial={{ chain_task_names: ['task-a'], chain_on_failure: false }}
      />
    );
    await user.click(screen.getByRole('combobox'));
    const option = await screen.findByRole('option', { name: 'task-a' });
    expect(option).toHaveAttribute('aria-disabled', 'true');
    const fresh = await screen.findByRole('option', { name: 'task-b' });
    expect(fresh).not.toHaveAttribute('aria-disabled', 'true');
  });

  it('disables the dropdown entirely when no selectable tasks remain', () => {
    render(
      <Harness
        availableTasks={[{ name: 'task-self' }]}
        currentTaskName="task-self"
      />
    );
    expect(screen.getByRole('combobox')).toHaveAttribute(
      'aria-disabled',
      'true'
    );
    expect(
      screen.getByText(/no tasks available to chain/i)
    ).toBeInTheDocument();
  });
});

describe('ChainBuilder remove', () => {
  it('removes a task when its remove button is clicked', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <Harness
        initial={{
          chain_task_names: ['task-a', 'task-b'],
          chain_on_failure: true,
        }}
        onChangeSpy={onChange}
      />
    );

    await user.click(screen.getByRole('button', { name: 'Remove task-a' }));

    expect(onChange).toHaveBeenLastCalledWith({
      chain_task_names: ['task-b'],
      chain_on_failure: true,
    });
  });

  it('removes only the clicked occurrence when names duplicate (stale data)', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <Harness
        initial={{
          chain_task_names: ['task-a', 'task-b', 'task-a'],
          chain_on_failure: false,
        }}
        onChangeSpy={onChange}
      />
    );

    // Two chips share the same accessible name "Remove task-a" — click the first.
    const removeButtons = screen.getAllByRole('button', {
      name: 'Remove task-a',
    });
    expect(removeButtons).toHaveLength(2);
    await user.click(removeButtons[0]);

    expect(onChange).toHaveBeenLastCalledWith({
      chain_task_names: ['task-b', 'task-a'],
      chain_on_failure: false,
    });
  });
});

describe('ChainBuilder reorder', () => {
  it('reorders chips via keyboard drag (move first chip right)', async () => {
    // jsdom returns zero rects, so dnd-kit's KeyboardSensor can't locate the
    // adjacent sortable. Stub getBoundingClientRect so chips lay out left-to-right.
    const order = ['task-a', 'task-b', 'task-c'];
    const original = HTMLElement.prototype.getBoundingClientRect;
    HTMLElement.prototype.getBoundingClientRect = function (): DOMRect {
      const handles = this.querySelectorAll('[aria-label^="Drag "]');
      if (handles.length === 1) {
        const name = (handles[0].getAttribute('aria-label') ?? '').slice(
          'Drag '.length
        );
        const i = order.indexOf(name);
        if (i >= 0) {
          const x = i * 100;
          return {
            x,
            y: 0,
            width: 80,
            height: 32,
            top: 0,
            left: x,
            right: x + 80,
            bottom: 32,
            toJSON: () => ({}),
          } as DOMRect;
        }
      }
      return {
        x: 0,
        y: 0,
        width: 0,
        height: 0,
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        toJSON: () => ({}),
      } as DOMRect;
    };

    try {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(
        <Harness
          initial={{
            chain_task_names: ['task-a', 'task-b', 'task-c'],
            chain_on_failure: false,
          }}
          onChangeSpy={onChange}
        />
      );

      const handle = screen.getByRole('button', { name: 'Drag task-a' });
      handle.focus();
      await user.keyboard('[Space]');
      await user.keyboard('[ArrowRight]');
      await user.keyboard('[Space]');

      expect(onChange).toHaveBeenLastCalledWith({
        chain_task_names: ['task-b', 'task-a', 'task-c'],
        chain_on_failure: false,
      });
    } finally {
      HTMLElement.prototype.getBoundingClientRect = original;
    }
  });
});

describe('ChainBuilder chain_on_failure toggle', () => {
  it('emits chain_on_failure=true when toggled on', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <Harness
        initial={{ chain_task_names: ['task-a'], chain_on_failure: false }}
        onChangeSpy={onChange}
      />
    );

    await user.click(screen.getByTestId('chain-on-failure-checkbox'));

    expect(onChange).toHaveBeenLastCalledWith({
      chain_task_names: ['task-a'],
      chain_on_failure: true,
    });
  });

  it('emits chain_on_failure=false when toggled off', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <Harness
        initial={{ chain_task_names: ['task-a'], chain_on_failure: true }}
        onChangeSpy={onChange}
      />
    );

    await user.click(screen.getByTestId('chain-on-failure-checkbox'));

    expect(onChange).toHaveBeenLastCalledWith({
      chain_task_names: ['task-a'],
      chain_on_failure: false,
    });
  });

  it('hides the toggle when the chain becomes empty after a remove', async () => {
    const user = userEvent.setup();
    render(
      <Harness
        initial={{ chain_task_names: ['task-a'], chain_on_failure: true }}
      />
    );
    expect(screen.getByTestId('chain-on-failure-checkbox')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Remove task-a' }));

    expect(
      screen.queryByTestId('chain-on-failure-checkbox')
    ).not.toBeInTheDocument();
  });
});

describe('ChainBuilder disabled', () => {
  it('disables drag, remove, dropdown, and failure toggle when disabled', () => {
    render(
      <Harness
        disabled
        initial={{ chain_task_names: ['task-a'], chain_on_failure: false }}
      />
    );
    expect(screen.getByRole('button', { name: 'Drag task-a' })).toBeDisabled();
    expect(
      screen.getByRole('button', { name: 'Remove task-a' })
    ).toBeDisabled();
    expect(screen.getByRole('combobox')).toHaveAttribute(
      'aria-disabled',
      'true'
    );
    expect(screen.getByTestId('chain-on-failure-checkbox')).toBeDisabled();
  });
});
