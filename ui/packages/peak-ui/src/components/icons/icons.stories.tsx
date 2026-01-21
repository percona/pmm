import { type Meta, type StoryObj } from '@storybook/react';
import {
  MongoIcon,
  MongoLeafIcon,
  MySqlDolphinIcon,
  MySqlIcon,
  PostgreSqlElephantIcon,
  PostgreSqlIcon,
} from './db';
import {
  EverestAppCircleIcon,
  EverestAppRoundIcon,
  EverestAppSquareIcon,
  EverestHorizontalAlternateIcon,
  EverestHorizontalIcon,
  EverestMainIcon,
} from './everest';
import { DatabaseIcon, NetworkNode } from './other';
import { GenericErrorIcon } from './generic-error';
import { NoMatchIcon } from './no-match';
import { EmptyStateIcon } from './empty-state';
import {
  ErrorIcon,
  PausedIcon,
  PendingIcon,
  SuccessIcon,
  UnknownIcon,
  WarningIcon,
} from './status';
import { Stack, Typography } from '@mui/material';
import * as DocBlock from '@storybook/blocks';

const meta = {
  title: 'Foundations/Icons',
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
    docs: {
      source: {
        code: null,
      },
      page: () => (
        <>
          <DocBlock.Title />
          <DocBlock.Subtitle />
          <DocBlock.Description />
          <DocBlock.Stories />
        </>
      ),
    },
  },
} satisfies Meta;

export default meta;

const icons = {
  db: [
    MySqlIcon,
    MongoIcon,
    PostgreSqlIcon,
    MySqlDolphinIcon,
    PostgreSqlElephantIcon,
    MongoLeafIcon,
  ],

  everest: [
    EverestMainIcon,
    EverestHorizontalIcon,
    EverestHorizontalAlternateIcon,
    EverestAppSquareIcon,
    EverestAppRoundIcon,
    EverestAppCircleIcon,
  ],

  genericError: [GenericErrorIcon],

  noMatch: [NoMatchIcon],

  emptyState: [EmptyStateIcon],

  status: [
    ErrorIcon,
    WarningIcon,
    PendingIcon,
    SuccessIcon,
    UnknownIcon,
    PausedIcon,
  ],
  other: [DatabaseIcon, NetworkNode],
};

export const Database: StoryObj<typeof MySqlIcon> = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
        <MySqlIcon fontSize='medium' />
        \`\`\``,
      },
    },
  },
  args: {
    fontSize: 'medium',
  },
  argTypes: {
    fontSize: {
      options: ['small', 'medium', 'large'],
      control: { type: 'inline-radio' },
    },
  },

  render: function Render({ fontSize }) {
    return (
      <>
        <Stack direction={'column'} rowGap={'2rem'}>
          {icons.db.map((Icon) => (
            <Stack direction={'row'} alignItems={'center'} columnGap={'2rem'}>
              <Icon fontSize={fontSize} />
              <Typography variant="body1">{Icon.name}</Typography>
            </Stack>
          ))}
        </Stack>
      </>
    );
  },
};

export const Everest: StoryObj<typeof EverestMainIcon> = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
        <EverestMainIcon fontSize='medium' />
        \`\`\``,
      },
    },
  },
  args: {
    fontSize: 'medium',
  },
  argTypes: {
    fontSize: {
      options: ['small', 'medium', 'large'],
      control: { type: 'inline-radio' },
    },
  },

  render: function Render({ fontSize }) {
    return (
      <Stack direction={'column'} rowGap={'2rem'}>
        {icons.everest.map((Icon) => (
          <Stack direction={'row'} alignItems={'center'} columnGap={'2rem'}>
            <Icon fontSize={fontSize} />
            <Typography variant="body1">{Icon.name}</Typography>
          </Stack>
        ))}
      </Stack>
    );
  },
};

export const GenericError: StoryObj<typeof GenericErrorIcon> = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
        <GenericErrorIcon h="128px" w="128px"  />
        \`\`\``,
      },
    },
  },

  render: function Render() {
    return (
      <Stack direction={'column'} rowGap={'2rem'}>
        {icons.genericError.map((Icon) => (
          <Stack direction={'row'} alignItems={'center'} columnGap={'2rem'}>
            <Icon h="128px" w="128px" />
            <Typography variant="body1">{Icon.name}</Typography>
          </Stack>
        ))}
      </Stack>
    );
  },
};

export const NoMatch: StoryObj<typeof NoMatchIcon> = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
        <NoMatchIcon h="128px" w="128px"  />
        \`\`\``,
      },
    },
  },

  render: function Render() {
    return (
      <Stack direction={'column'} rowGap={'2rem'}>
        {icons.noMatch.map((Icon) => (
          <Stack direction={'row'} alignItems={'center'} columnGap={'2rem'}>
            <Icon h="128px" w="128px" />
            <Typography variant="body1">{Icon.name}</Typography>
          </Stack>
        ))}
      </Stack>
    );
  },
};

export const EmptyState: StoryObj<typeof EmptyStateIcon> = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
        <EmptyState />
        \`\`\``,
      },
    },
  },

  render: function Render() {
    return (
      <Stack direction={'column'} rowGap={'2rem'}>
        {icons.emptyState.map((Icon) => (
          <Stack direction={'row'} alignItems={'center'} columnGap={'2rem'}>
            <Icon h="60px" w="60px" />
            <Typography variant="body1">{Icon.name}</Typography>
          </Stack>
        ))}
      </Stack>
    );
  },
};

export const Status: StoryObj<typeof ErrorIcon> = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
        <ErrorIcon />
        \`\`\``,
      },
    },
  },

  render: function Render() {
    return (
      <Stack direction={'column'} rowGap={'2rem'}>
        {icons.status.map((Icon) => (
          <Stack direction={'row'} alignItems={'center'} columnGap={'2rem'}>
            <Icon />
            <Typography variant="body1">{Icon.name}</Typography>
          </Stack>
        ))}
      </Stack>
    );
  },
};

export const Other: StoryObj<typeof DatabaseIcon> = {
  parameters: {
    docs: {
      description: {
        story: `\`\`\`ts
        <DatabaseIcon />
        \`\`\``,
      },
    },
  },

  render: function Render() {
    return (
      <Stack direction={'column'} rowGap={'2rem'}>
        {icons.other.map((Icon) => (
          <Stack direction={'row'} alignItems={'center'} columnGap={'2rem'}>
            <Icon />
            <Typography variant="body1">{Icon.name}</Typography>
          </Stack>
        ))}
      </Stack>
    );
  },
};
