import type { Meta, StoryObj } from '@storybook/react';
import { FormProvider, useForm } from 'react-hook-form';
import { TextArrayProps } from './text-array.types';
import TextArray from './text-array';

const meta = {
  title: 'TextArray',
  component: TextArray,
  parameters: {
    layout: 'centered',
  },

  argTypes: {
    label: {
      type: 'string',
    },
    placeholder: {
      type: 'string',
    },
  },

  render: function Render({ label, placeholder }) {
    const methods = useForm({
      defaultValues: {
        textArrayValues: [
          {
            textArrayValue: 'first',
          },
          {
            textArrayValue: 'second',
          },
          {
            textArrayValue: 'third',
          },
        ],
      },
    });

    return (
      <FormProvider {...methods}>
        <TextArray
          fieldName="textArrayValues"
          fieldKey="textArrayValue"
          label={label}
          placeholder={placeholder}
        />
      </FormProvider>
    );
  },
} satisfies Meta<TextArrayProps>;

export default meta;

type Story = StoryObj<TextArrayProps>;

export const Basic: Story = {
  args: {
    label: 'Label',
    placeholder: 'Placeholder',
  },
};
