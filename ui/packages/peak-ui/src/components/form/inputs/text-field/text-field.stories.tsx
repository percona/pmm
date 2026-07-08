import { type Meta, type StoryObj } from '@storybook/react';
import { useState, type ReactNode } from 'react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import InputAdornment from '@mui/material/InputAdornment';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import type { TextFieldProps } from '@mui/material/TextField';
import AlternateEmailOutlined from '@mui/icons-material/AlternateEmailOutlined';
import AttachMoneyOutlined from '@mui/icons-material/AttachMoneyOutlined';
import CloseOutlined from '@mui/icons-material/CloseOutlined';
import InfoOutlined from '@mui/icons-material/InfoOutlined';
import SearchOutlined from '@mui/icons-material/SearchOutlined';
import VisibilityOutlined from '@mui/icons-material/VisibilityOutlined';
import VisibilityOffOutlined from '@mui/icons-material/VisibilityOffOutlined';

const SAMPLE_SELECT_OPTIONS = [
  { value: 'production', label: 'Production' },
  { value: 'staging', label: 'Staging' },
  { value: 'development', label: 'Development' },
];

type StartAdornmentChoice = 'none' | 'search' | 'currency' | 'at' | 'info';
type EndAdornmentChoice = 'none' | 'clear' | 'visibility' | 'info';

type PlaygroundArgs = TextFieldProps & {
  startAdornment?: StartAdornmentChoice;
  endAdornment?: EndAdornmentChoice;
};

const startAdornmentMap: Record<StartAdornmentChoice, ReactNode> = {
  none: null,
  search: <SearchOutlined fontSize="large" />,
  currency: <AttachMoneyOutlined fontSize="medium" />,
  at: <AlternateEmailOutlined fontSize="small" />,
  info: <InfoOutlined />,
};

const endAdornmentMap: Record<EndAdornmentChoice, ReactNode> = {
  none: null,
  clear: (
    <Tooltip title="Clear">
      <IconButton size="small">
        <CloseOutlined />
      </IconButton>
    </Tooltip>
  ),
  visibility: (
    <Tooltip title="Toggle visibility">
      <IconButton size="large">
        <VisibilityOutlined />
      </IconButton>
    </Tooltip>
  ),
  info: (
    <Tooltip title="More info">
      <IconButton size="medium">
        <InfoOutlined />
      </IconButton>
    </Tooltip>
  ),
};

const meta = {
  title: 'Inputs/Text Field',
  component: TextField,
  tags: ['autodocs', 'stable'],
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: [
          'The design-system-styled MUI `TextField` primitive. For react-hook-form usage, see the **TextInput** wrapper component.',
        ].join('\n'),
      },
      page: () => (
        <>
          <DocBlock.Title />
          <DocBlock.Subtitle />
          <DocBlock.Description />
          <DocBlock.Primary />
          <DocBlock.Controls />
          <DocBlock.Stories includePrimary={false} />
        </>
      ),
    },
  },
  argTypes: {
    size: {
      options: ['small', 'medium'],
      control: { type: 'inline-radio' },
      description: 'Controls field height — small is 40 px, medium is 56 px.',
      table: { category: 'Size', defaultValue: { summary: "'medium'" } },
    },
    label: {
      control: 'text',
      description: 'Label text displayed above or inside the field.',
      table: { category: 'Content' },
    },
    placeholder: {
      control: 'text',
      description: 'Placeholder shown when the field is empty.',
      table: { category: 'Content' },
    },
    defaultValue: {
      control: 'text',
      description:
        'Initial value — useful to preview the filled-in state and how the label shrinks.',
      table: { category: 'Content' },
    },
    helperText: {
      control: 'text',
      description:
        'Assistive text rendered below the field. Pass a single space `" "` to reserve vertical space without showing text (prevents layout shift when an error appears).',
      table: { category: 'Content' },
    },
    type: {
      options: ['text', 'number', 'password', 'email', 'tel', 'url'],
      control: 'select',
      description: 'HTML input type.',
      table: { category: 'Content', defaultValue: { summary: "'text'" } },
    },
    select: {
      control: 'boolean',
      description:
        'Renders the field as a select (dropdown caret on the right). Real usage requires `<MenuItem>` children. Can be combined with an `endAdornment`.',
      table: { category: 'Content', defaultValue: { summary: 'false' } },
    },
    startAdornment: {
      options: ['none', 'search', 'currency', 'at', 'info'],
      control: { type: 'select' },
      description: 'Plain icon shown in the start position of the field.',
      table: { category: 'Adornments', defaultValue: { summary: "'none'" } },
    },
    endAdornment: {
      options: ['none', 'clear', 'visibility', 'info'],
      control: { type: 'select' },
      description:
        'Icon button shown in the end position. Demonstrates the 24 px layout / 40 px tap-area bleed.',
      table: { category: 'Adornments', defaultValue: { summary: "'none'" } },
    },
    multiline: {
      control: 'boolean',
      description:
        'Render as a textarea. The field grows with content between `minRows` and `maxRows`.',
      table: { category: 'Layout', defaultValue: { summary: 'false' } },
    },
    fullWidth: {
      control: 'boolean',
      description: 'Stretch the field to the full width of its container.',
      table: { category: 'Layout', defaultValue: { summary: 'false' } },
    },
    focused: {
      control: 'boolean',
      description:
        'Force the focused visual state — useful for previewing the focus ring without clicking the field.',
      table: { category: 'State', defaultValue: { summary: 'false' } },
    },
    error: {
      control: 'boolean',
      description: 'Applies the error palette to border, label, and helper text.',
      table: { category: 'State', defaultValue: { summary: 'false' } },
    },
    disabled: {
      control: 'boolean',
      description: 'Disables the field.',
      table: { category: 'State', defaultValue: { summary: 'false' } },
    },
    required: {
      control: 'boolean',
      description: 'Marks the field as required (adds asterisk to the label).',
      table: { category: 'State', defaultValue: { summary: 'false' } },
    },
  },
} satisfies Meta<PlaygroundArgs>;

export default meta;
type Story = StoryObj<PlaygroundArgs>;

export const Playground: Story = {
  parameters: {
    docs: {
      source: {
        code: '// See the named stories below (With start/end adornment, Password with toggle, Multiline, etc.) for concrete usage patterns.',
      },
    },
  },
  args: {
    size: 'small',
    label: 'Label',
    placeholder: 'Placeholder',
    defaultValue: '',
    helperText: 'Helper text',
    type: 'text',
    select: false,
    startAdornment: 'none',
    endAdornment: 'none',
    multiline: false,
    fullWidth: true,
    focused: false,
    error: false,
    disabled: false,
    required: false,
  },
  render: (args) => {
    const { startAdornment, endAdornment, multiline, focused, select, ...rest } = args;
    const startKey: StartAdornmentChoice = startAdornment ?? 'none';
    const endKey: EndAdornmentChoice = endAdornment ?? 'none';
    const startNode = startAdornmentMap[startKey];
    const endNode = endAdornmentMap[endKey];
    return (
      <Box sx={{ width: 320 }}>
        <TextField
          variant="outlined"
          multiline={!select && multiline}
          minRows={!select && multiline ? 3 : undefined}
          maxRows={!select && multiline ? 8 : undefined}
          focused={focused || undefined}
          select={select}
          {...rest}
          slotProps={{
            input: {
              startAdornment: startNode ? (
                <InputAdornment position="start">{startNode}</InputAdornment>
              ) : undefined,
              endAdornment: endNode ? (
                <InputAdornment position="end">{endNode}</InputAdornment>
              ) : undefined,
            },
          }}
        >
          {select
            ? SAMPLE_SELECT_OPTIONS.map((option) => (
                <MenuItem key={option.value} value={option.value}>
                  {option.label}
                </MenuItem>
              ))
            : undefined}
        </TextField>
      </Box>
    );
  },
};

export const Sizes: Story = {
  parameters: {
    docs: {
      description: {
        story:
          'Small fields are 40 px tall; medium fields are 56 px. Both share the same styling, border radius, horizontal padding, etc.',
      },
    },
  },
  render: () => (
    <Stack direction="row" gap={3} alignItems="center">
      <Stack gap={1} alignItems="center">
        <TextField size="small" label="Small" placeholder="40 px" variant="outlined" />
      </Stack>
      <Stack gap={1} alignItems="center">
        <TextField size="medium" label="Medium" placeholder="56 px" variant="outlined" />
      </Stack>
    </Stack>
  ),
};

export const States: Story = {
  parameters: {
    docs: {
      description: {
        story:
          'Default, error, and disabled states side by side. The error state uses the error palette for the border, label, and helper text.',
      },
    },
  },
  render: () => (
    <Stack direction="row" gap={3} alignItems="flex-start">
      <TextField
        size="small"
        label="Default"
        defaultValue="Value"
        helperText="Helper text"
        variant="outlined"
      />
      <TextField
        size="small"
        label="Error"
        defaultValue="Invalid"
        helperText="Something went wrong"
        error
        variant="outlined"
      />
      <TextField
        size="small"
        label="Disabled"
        defaultValue="Read only"
        helperText="Cannot edit"
        disabled
        variant="outlined"
      />
    </Stack>
  ),
};

export const FilledIn: Story = {
  name: 'Filled in vs empty',
  parameters: {
    docs: {
      description: {
        story:
          'Compares empty and filled states. When filled, the label shrinks and the value text uses `text.primary`.',
      },
    },
  },
  render: () => (
    <Stack direction="row" gap={3} alignItems="flex-start">
      <TextField size="small" label="Empty" placeholder="Type here…" variant="outlined" />
      <TextField size="small" label="Filled in" defaultValue="percona-server" variant="outlined" />
    </Stack>
  ),
};

export const WithStartAdornment: Story = {
  name: 'With start adornment',
  parameters: {
    docs: {
      description: {
        story: 'A plain icon in the start position.',
      },
    },
  },
  render: () => (
    <Stack gap={3} sx={{ width: 280 }}>
      <TextField
        size="small"
        label="Search"
        placeholder="Search…"
        variant="outlined"
        slotProps={{
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <SearchOutlined fontSize="large" />
              </InputAdornment>
            ),
          },
        }}
      />
      <TextField
        size="medium"
        label="Search"
        placeholder="Search…"
        variant="outlined"
        slotProps={{
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <SearchOutlined fontSize="large" />
              </InputAdornment>
            ),
          },
        }}
      />
    </Stack>
  ),
};

export const WithEndAdornment: Story = {
  name: 'With end adornment',
  parameters: {
    docs: {
      description: {
        story:
          'An `IconButton` in the end position. The button occupies 24 px of layout space but keeps a 40 px tap area — it bleeds 8 px on each side.',
      },
    },
  },
  render: function Render() {
    const [value, setValue] = useState('percona');
    return (
      <Stack gap={3} sx={{ width: 280 }}>
        <TextField
          size="small"
          label="Clearable"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          variant="outlined"
          slotProps={{
            input: {
              endAdornment: value ? (
                <InputAdornment position="end">
                  <Tooltip title="Clear">
                    <IconButton size="small" onClick={() => setValue('')}>
                      <CloseOutlined />
                    </IconButton>
                  </Tooltip>
                </InputAdornment>
              ) : null,
            },
          }}
        />
      </Stack>
    );
  },
};

export const WithBothAdornments: Story = {
  name: 'With both adornments',
  parameters: {
    docs: {
      description: {
        story:
          'Icon on the left, icon button on the right. Shows how both adornment positions work together.',
      },
    },
  },
  render: function Render() {
    const [value, setValue] = useState('percona');
    return (
      <Stack gap={3} sx={{ width: 280 }}>
        <TextField
          size="small"
          label="Search"
          placeholder="Search…"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          variant="outlined"
          slotProps={{
            input: {
              startAdornment: (
                <InputAdornment position="start">
                  <SearchOutlined fontSize="large" />
                </InputAdornment>
              ),
              endAdornment: value ? (
                <InputAdornment position="end">
                  <Tooltip title="Clear">
                    <IconButton size="small" onClick={() => setValue('')}>
                      <CloseOutlined />
                    </IconButton>
                  </Tooltip>
                </InputAdornment>
              ) : null,
            },
          }}
        />
      </Stack>
    );
  },
};

export const NumberInput: Story = {
  name: 'Number input',
  parameters: {
    docs: {
      description: {
        story:
          'A `type="number"` field with `min`, `max`, and `step` constraints via `slotProps.htmlInput`.',
      },
    },
  },
  render: () => (
    <Box sx={{ width: 280 }}>
      <TextField
        size="small"
        label="Retention (days)"
        type="number"
        defaultValue={30}
        variant="outlined"
        helperText="Value between 1 and 365"
        slotProps={{
          htmlInput: { min: 1, max: 365, step: 1 },
        }}
      />
    </Box>
  ),
};

export const PasswordWithToggle: Story = {
  name: 'Password with toggle',
  parameters: {
    docs: {
      description: {
        story:
          'A password field with a visibility toggle in the end adornment. Demonstrates the icon button bleed behavior with a functional use case.',
      },
    },
  },
  render: function Render() {
    const [visible, setVisible] = useState(false);
    return (
      <Box sx={{ width: 280 }}>
        <TextField
          size="small"
          label="Password"
          type={visible ? 'text' : 'password'}
          defaultValue="supersecret"
          variant="outlined"
          slotProps={{
            input: {
              endAdornment: (
                <InputAdornment position="end">
                  <Tooltip title={visible ? 'Hide password' : 'Show password'}>
                    <IconButton size="small" onClick={() => setVisible((v) => !v)}>
                      {visible ? <VisibilityOffOutlined /> : <VisibilityOutlined />}
                    </IconButton>
                  </Tooltip>
                </InputAdornment>
              ),
            },
          }}
        />
      </Box>
    );
  },
};

export const Multiline: Story = {
  parameters: {
    docs: {
      description: {
        story:
          'Textarea variant with `multiline`, `minRows`, and `maxRows`. Used in PMM for SSH keys and multi-line configuration values.',
      },
    },
  },
  render: () => (
    <Box sx={{ width: 360 }}>
      <TextField
        label="SSH Public Key"
        placeholder="Paste your public key here…"
        multiline
        minRows={3}
        maxRows={8}
        variant="outlined"
        helperText="Supports RSA, ECDSA, and Ed25519 keys"
        fullWidth
      />
    </Box>
  ),
};

export const WithSelect: Story = {
  name: 'As a select',
  parameters: {
    docs: {
      description: {
        story: [
          'Pass `select` to render the field as a dropdown. The caret icon is provided by MUI — pass `<MenuItem>` children for the options.',
          '',
          'Pairing with an `endAdornment` (e.g. a clear button) is supported.',
        ].join('\n'),
      },
    },
  },
  render: function Render() {
    const [plain, setPlain] = useState('');
    const [withClear, setWithClear] = useState('staging');
    return (
      <Stack gap={3} sx={{ width: 280 }}>
        <TextField
          select
          size="small"
          label="Environment"
          placeholder="Select an environment"
          value={plain}
          onChange={(e) => setPlain(e.target.value)}
          variant="outlined"
          slotProps={{
            select: {
              displayEmpty: true,
              renderValue: (value) =>
                value ? (
                  SAMPLE_SELECT_OPTIONS.find((o) => o.value === value)?.label
                ) : (
                  <Box sx={{ opacity: 0.5 }}>Select an environment</Box>
                ),
            },
            inputLabel: { shrink: true },
          }}
        >
          {SAMPLE_SELECT_OPTIONS.map((option) => (
            <MenuItem key={option.value} value={option.value}>
              {option.label}
            </MenuItem>
          ))}
        </TextField>
        <TextField
          select
          size="small"
          label="Environment (clearable)"
          placeholder="Select an environment"
          value={withClear}
          onChange={(e) => setWithClear(e.target.value)}
          variant="outlined"
          slotProps={{
            select: {
              displayEmpty: true,
              renderValue: (value) =>
                value ? (
                  SAMPLE_SELECT_OPTIONS.find((o) => o.value === value)?.label
                ) : (
                  <Box sx={{ opacity: 0.5 }}>Select an environment</Box>
                ),
            },
            inputLabel: { shrink: true },
            input: {
              endAdornment: withClear ? (
                <InputAdornment position="end">
                  <Tooltip title="Clear">
                    <IconButton
                      size="small"
                      onClick={(event) => {
                        event.stopPropagation();
                        setWithClear('');
                      }}
                    >
                      <CloseOutlined />
                    </IconButton>
                  </Tooltip>
                </InputAdornment>
              ) : null,
            },
          }}
        >
          {SAMPLE_SELECT_OPTIONS.map((option) => (
            <MenuItem key={option.value} value={option.value}>
              {option.label}
            </MenuItem>
          ))}
        </TextField>
      </Stack>
    );
  },
};

export const LabelInline: Story = {
  name: 'Label inline',
  parameters: {
    docs: {
      description: {
        story:
          'Shows the label behavior in both positions: floating inside the field (unfocused empty) and shrunk above the border (focused or filled). The shrunk label uses `fontWeight: 500` and `letterSpacing: 0.16px`.',
      },
    },
  },
  render: () => (
    <Stack direction="row" gap={3} alignItems="flex-start">
      <TextField size="small" label="Unfocused empty" variant="outlined" />
      <TextField size="small" label="Focused / filled" defaultValue="percona" variant="outlined" />
    </Stack>
  ),
};

export const WithHelperText: Story = {
  name: 'With helper text',
  parameters: {
    docs: {
      description: {
        story:
          'Helper text rendered below the field. Uses `typography.helperText` (12 px, weight 450, line-height 1.25, letter-spacing 0.12px). Error helper text inherits the error color.',
      },
    },
  },
  render: () => (
    <Stack direction="row" gap={3} alignItems="flex-start">
      <TextField size="small" label="Normal" helperText="Helpful guidance" variant="outlined" />
      <TextField
        size="small"
        label="Error"
        helperText="This field is required"
        error
        variant="outlined"
      />
    </Stack>
  ),
};
