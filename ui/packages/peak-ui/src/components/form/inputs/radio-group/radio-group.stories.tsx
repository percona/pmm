import type { Meta, StoryObj } from '@storybook/react';
import { FormProvider, useForm } from 'react-hook-form';

import RadioGroup from './radio-group';
import { RadioGroupProps } from './radio-group.types';

type ExpandedRadioGroupProps = RadioGroupProps & {
  customSize: number;
  required: boolean;
};

const meta = {
  title: 'RadioGroup',
  component: RadioGroup,
  parameters: {
    layout: 'centered',
  },
  render: function Render({ required, customSize, options, ...args }) {
    const methods = useForm();

    return (
      <FormProvider {...methods}>
        <RadioGroup
          {...args}
          options={options}
          name="RadioGroup1"
          label="Small size"
          isRequired={required}
          radioGroupFieldProps={{
            sx: {
              '& .MuiSvgIcon-root': {
                fontSize: 20,
              },
            },
          }}
          controllerProps={{
            defaultValue: 'First',
          }}
        />

        <RadioGroup
          {...args}
          options={options}
          name="RadioGroup2"
          label="Medium size"
          isRequired={required}
          radioGroupFieldProps={{
            sx: {
              '& .MuiSvgIcon-root': {
                fontSize: 24,
              },
            },
          }}
          controllerProps={{
            defaultValue: 'First',
          }}
        />

        <RadioGroup
          {...args}
          options={options}
          name="RadioGroup3"
          label="Custom size(the default size is large)"
          isRequired={required}
          radioGroupFieldProps={{
            sx: {
              '& .MuiSvgIcon-root': {
                fontSize: customSize,
              },
            },
          }}
          controllerProps={{
            defaultValue: 'First',
          }}
        />
      </FormProvider>
    );
  },
} satisfies Meta<ExpandedRadioGroupProps>;
export default meta;

type Story = StoryObj<ExpandedRadioGroupProps>;

export const Basic: Story = {
  args: {
    options: [
      { label: 'First', value: 'First', disabled: false },
      { label: 'Second', value: 'Second', disabled: false },
      { label: 'Third', value: 'Third', disabled: false },
      { label: 'Fourth', value: 'Fourth', disabled: true },
    ],
    customSize: 28,
    required: false,
  },
};
