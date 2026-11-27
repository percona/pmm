import type { Meta, StoryObj } from '@storybook/react';
import { FormProvider, useForm } from 'react-hook-form';
import { ToggleButtonGroupInputProps } from '../toggle-button-group/toggle-button-group.types';
import ToggleButtonGroupInputRegular from './toggle-button-group-regular';
import ToggleRegularButton from '../../../buttons/toggle-regular-button';

type CustomArgs = ToggleButtonGroupInputProps & {
  disabled?: boolean;
  fullWidth?: boolean;
  size?: 'small' | 'medium' | 'large';
};

const meta = {
  title: 'To be reviewed/ToggleButtonGroupInputRegular',
  component: ToggleButtonGroupInputRegular,
  tags: ['needs-review'],
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
    fullWidth: {
      type: 'boolean',
    },
    size: {
      options: ['small', 'medium', 'large'],
      control: { type: 'inline-radio' },
    },
  },

  render: function Render({ label, disabled, fullWidth, size }) {
    const methods = useForm();

    return (
      <FormProvider {...methods}>
        <ToggleButtonGroupInputRegular
          name="view-mode"
          label={label}
          toggleButtonGroupProps={{ disabled, fullWidth, size }}
          controllerProps={{
            defaultValue: 'table',
          }}
        >
          <ToggleRegularButton value="table">Table</ToggleRegularButton>
          <ToggleRegularButton value="graph">Graph</ToggleRegularButton>
          <ToggleRegularButton value="raw">Raw data</ToggleRegularButton>
        </ToggleButtonGroupInputRegular>
      </FormProvider>
    );
  },
} satisfies Meta<CustomArgs>;

export default meta;

type Story = StoryObj<CustomArgs>;

export const Basic: Story = {
  args: {
    label: 'View mode',
    disabled: false,
    fullWidth: false,
    size: 'small',
  },
};
