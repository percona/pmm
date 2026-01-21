import type { Meta, StoryObj } from '@storybook/react';
import { FormProvider, useForm } from 'react-hook-form';
import { ToggleButtonGroupInputProps } from './toggle-button-group.types';
import ToggleButtonGroupInput from './toggle-button-group';
import ToggleCard from '../../../buttons/toggle-card';

type CustomArgs = ToggleButtonGroupInputProps & {
  disabled?: boolean;
  size?: 'small' | 'medium' | 'large';
  width?: string;
};

const meta = {
  title: 'Inputs/ToggleButtonGroupInput',
  component: ToggleButtonGroupInput,
  parameters: {
    layout: 'centered',
  },

  argTypes: {
    label: {
      type: 'string',
    },
    disabled: {
      type: 'boolean',
    },
    size: {
      options: ['small', 'medium', 'large'],
      control: { type: 'inline-radio' },
    },
    width: {
      type: 'string',
    },
  },

  render: function Render({ label, disabled, size, width }) {
    const methods = useForm();

    return (
      <FormProvider {...methods}>
        <ToggleButtonGroupInput
          name="toggle-button-group-input"
          label={label}
          toggleButtonGroupProps={{ disabled, size, sx: { width } }}
          controllerProps={{
            name: 'toggle-button-group-input',
            defaultValue: 'small',
          }}
        >
          <ToggleCard value={'small'}> small </ToggleCard>
          <ToggleCard value={'medium'}> medium </ToggleCard>
          <ToggleCard value={'large'}> large </ToggleCard>
        </ToggleButtonGroupInput>
      </FormProvider>
    );
  },
} satisfies Meta<CustomArgs>;

export default meta;

type Story = StoryObj<CustomArgs>;

export const Basic: Story = {
  args: {
    label: 'Resource size',
    disabled: false,
    size: 'medium',
    width: '50rem',
  },
};
