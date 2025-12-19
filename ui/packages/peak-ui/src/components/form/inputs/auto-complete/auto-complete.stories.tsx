import type { Meta, StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/blocks';
import { FormProvider, useForm } from 'react-hook-form';
import AutoCompleteInput from './auto-complete';

type CustomArgs = React.ComponentProps<typeof AutoCompleteInput> & {
  error?: boolean;
  size?: 'small' | 'medium';
  width?: string;
  helperText?: string;
};

const meta = {
  title: 'AutoComplete',
  component: AutoCompleteInput,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
    docs: {
      source: null,
    },
    toc: true,
    page: () => (
      <>
        <DocBlock.Title />
        <DocBlock.Subtitle />
        <DocBlock.Description />
        <DocBlock.Source
          code={`
<AutoCompleteInput
  textFieldProps={{
    error: args.error,
    size: args.size,
    helperText: args.helperText,
  }}
  autoCompleteProps={{
    sx: { width: args.width },
    defaultValue: args.options[0],
  }}
  name={'autocomplete'}
  label={args.label}
  options={args.options}
  disabled={args.disabled}
  loading={args.loading}
  isRequired={args.isRequired}
/>
            `}
        />
        <DocBlock.Primary />
        <DocBlock.Controls />
        <DocBlock.Stories />
      </>
    ),
  },

  argTypes: {
    disabled: {
      type: 'boolean',
    },
    isRequired: {
      type: 'boolean',
    },
    loading: {
      type: 'boolean',
    },
    error: {
      type: 'boolean',
    },
    size: {
      options: ['small', 'medium'],
      control: { type: 'inline-radio' },
    },
  },

  render: function Render(args) {
    const methods = useForm();

    return (
      <FormProvider {...methods}>
        <AutoCompleteInput
          textFieldProps={{
            error: args.error,
            size: args.size,
            helperText: args.helperText,
          }}
          autoCompleteProps={{
            sx: { width: args.width },
            defaultValue: args.options[0],
          }}
          name={'autocomplete'}
          label={args.label}
          options={args.options}
          disabled={args.disabled}
          loading={args.loading}
          isRequired={args.isRequired}
        />
      </FormProvider>
    );
  },
} satisfies Meta<CustomArgs>;
export default meta;

type Story = StoryObj<CustomArgs>;

export const Basic: Story = {
  tags: ['!autodocs'],
  args: {
    label: 'Label',
    disabled: false,
    isRequired: false,
    loading: false,
    error: false,
    size: 'small',
    options: ['First', 'Second', 'Third'],
    width: '200px',
    helperText: 'Select one of the options',
  },
};

export const Small: Story = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
          <AutoCompleteInput textFieldProps={{size: 'small'}} />
        \`\`\``,
      },
    },
  },
  args: {
    label: 'Small',
    size: 'small',
    options: ['First', 'Second', 'Third'],
    width: '200px',
  },
};

export const Medium: Story = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
          <AutoCompleteInput textFieldProps={{size: 'medium'}} />
        \`\`\``,
      },
    },
  },
  args: {
    label: 'Medium',
    size: 'medium',
    options: ['First', 'Second', 'Third'],
    width: '200px',
  },
};

export const Required: Story = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
          <AutoCompleteInput isRequired={true} />
        \`\`\``,
      },
    },
  },
  args: {
    label: 'Required',
    size: 'small',
    isRequired: true,
    options: ['First', 'Second', 'Third'],
    width: '200px',
  },
};

export const Disabled: Story = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
          <AutoCompleteInput disabled={true} />
        \`\`\``,
      },
    },
  },
  args: {
    label: 'Disabled',
    size: 'small',
    disabled: true,
    options: ['First', 'Second', 'Third'],
    width: '200px',
  },
};

export const Error: Story = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
          <AutoCompleteInput textFieldProps={
            {
              error: true,
              helperText: 'Error helper text'
            }
          } />
        \`\`\``,
      },
    },
  },
  args: {
    label: 'Error with helper',
    size: 'small',
    error: true,
    helperText: 'Error helper text',
    options: ['First', 'Second', 'Third'],
    width: '200px',
  },
};

export const Loading: Story = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
          <AutoCompleteInput loading={true} />
        \`\`\``,
      },
    },
  },
  args: {
    label: 'Loading',
    loading: true,
    size: 'small',
    options: ['First', 'Second', 'Third'],
    width: '200px',
  },
};

export const Hover: Story = {
  parameters: {
    pseudo: {
      hover: true,
    },
  },
  args: {
    label: 'Hover',
    size: 'small',
    options: ['First', 'Second', 'Third'],
    width: '200px',
  },
};

export const Focus: Story = {
  parameters: {
    pseudo: {
      focus: true,
    },
  },
  args: {
    label: 'Focus',
    size: 'small',
    options: ['First', 'Second', 'Third'],
    width: '200px',
  },
};
