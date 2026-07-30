import { CodeBlock as PerconaCodeBlock } from '@percona/percona-ui';
import { FC, PropsWithChildren } from 'react';

type Props = PropsWithChildren<{
  className?: string;
}>;

export const CodeBlock: FC<Props> = ({ children, className }) => {
  const isInline =
    typeof children === 'string' && !children.includes('\n') && !className;

  if (isInline) {
    // Native <code> gets the Peak Design inline code styling from the theme
    return <code>{children}</code>;
  }

  const language = /language-([^\s]+)/.exec(className ?? '')?.[1];
  const text = Array.isArray(children)
    ? children.join('')
    : String(children ?? '');

  return (
    <PerconaCodeBlock
      content={text.replace(/\n$/, '')}
      language={language}
      sx={{ my: 2 }}
    />
  );
};
