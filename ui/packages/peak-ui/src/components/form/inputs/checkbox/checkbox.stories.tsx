import type { Meta, StoryObj } from '@storybook/react';
import { FormProvider, useForm } from 'react-hook-form';
import Checkbox from './checkbox';
import * as DocBlock from '@storybook/blocks';

type CustomArgs = {
  label?: string;
  disabled?: boolean;
  size?: 'small' | 'medium';
  required?: boolean;
  indeterminate?: boolean;
  defaultChecked?: boolean;
};

const meta = {
  title: 'Inputs/Checkbox',
  tags: ['autodocs'],
  argTypes: {
    label: {
      defaultValue: '',
      type: 'string',
    },
    disabled: {
      defaultValue: false,
      type: 'boolean',
    },
    size: {
      options: ['small', 'medium'],
      defaultValue: 'medium',
      control: { type: 'inline-radio' },
    },
    required: {
      defaultValue: false,
      type: 'boolean',
    },
    indeterminate: {
      defaultValue: false,
      type: 'boolean',
    },
  },
  parameters: {
    layout: 'centered',
    docs: {
      toc: true,
      source: {
        code: null,
      },
      page: () => (
        <>
          <DocBlock.Title />
          <DocBlock.Subtitle />
          <DocBlock.Description />
          <DocBlock.Source
            code={`
          <Checkbox
  name={'custom'}
  label={'Custom'}
  checkboxProps={{ disabled: false, size: 'medium', indeterminate: false }}
  labelProps={{ isRequired: false }}
/>
            `}
          />
          <DocBlock.Primary />
          <DocBlock.Controls />
          <DocBlock.Stories />
        </>
      ),
    },
  },
  render: function Render({
    label,
    disabled,
    size,
    required,
    indeterminate,
    defaultChecked,
  }) {
    const methods = useForm();

    return (
      <FormProvider {...methods}>
        <Checkbox
          name="custom"
          label={label}
          checkboxProps={{ disabled, size, indeterminate, defaultChecked }}
          labelProps={{ isRequired: required }}
        />
      </FormProvider>
    );
  },
} satisfies Meta<CustomArgs>;

export default meta;

type Story = StoryObj<CustomArgs>;

export const Custom: Story = {
  tags: ['!autodocs'],
  args: {
    label: 'Custom',
    disabled: false,
    size: 'medium',
    required: false,
    indeterminate: false,
  },
};

export const Small: Story = {
  args: {
    label: 'Small',
    size: 'small',
  },
};

export const Medium: Story = {
  args: {
    label: 'Medium',
    size: 'medium',
  },
};

export const Checked: Story = {
  args: {
    label: 'Checked',
    defaultChecked: true,
  },
};

export const Disabled: Story = {
  args: {
    label: 'Disabled',
    disabled: true,
  },
};

export const Indeterminate: Story = {
  args: {
    label: 'Indeterminate',
    indeterminate: true,
  },
};
