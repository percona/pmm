import {
  ChangeEvent,
  CSSProperties,
  forwardRef,
  useImperativeHandle,
  useRef,
} from 'react';
import { InputBaseComponentProps } from '@mui/material/InputBase';
import { useTheme } from '@mui/material/styles';
import { Highlight, themes } from 'prism-react-renderer';
import Editor from 'react-simple-code-editor';

// Starlark is syntactically Python, so the python grammar fits check scripts.
const LANGUAGE = 'python';

// ScriptEditorInput adapts react-simple-code-editor (a highlighted <pre> under a
// transparent <textarea>) to MUI's InputBase `inputComponent` contract, so the
// Script field keeps its outlined look, label and react-hook-form wiring.
export const ScriptEditorInput = forwardRef<
  HTMLTextAreaElement,
  InputBaseComponentProps
>(function ScriptEditorInput(props, ref) {
  const {
    value,
    onChange,
    onFocus,
    onBlur,
    disabled,
    readOnly,
    placeholder,
    name,
    className,
    ...rest
  } = props;
  const theme = useTheme();
  // same scheme the percona-ui CodeBlock resolves per color mode
  const prismTheme =
    theme.palette.mode === 'dark' ? themes.okaidia : themes.nightOwlLight;
  // the editor's own ref only exposes its undo history, so the textarea is
  // reached through a layout-neutral wrapper instead
  const wrapperRef = useRef<HTMLDivElement | null>(null);

  // InputBase drives focus through the input ref (e.g. clicking the label)
  useImperativeHandle(
    ref,
    () =>
      ({
        focus: () => wrapperRef.current?.querySelector('textarea')?.focus(),
        get value() {
          return wrapperRef.current?.querySelector('textarea')?.value ?? '';
        },
      }) as unknown as HTMLTextAreaElement,
    []
  );

  return (
    <div ref={wrapperRef} style={{ display: 'contents' }}>
      <Editor
        value={typeof value === 'string' ? value : ''}
        onValueChange={(code) =>
          onChange?.({
            target: { value: code, name },
          } as unknown as ChangeEvent<HTMLTextAreaElement>)
        }
        highlight={(code) => (
          <Highlight code={code} language={LANGUAGE} theme={prismTheme}>
            {({ tokens, getLineProps, getTokenProps }) => (
              <>
                {tokens.map((line, i) => (
                  <div key={i} {...getLineProps({ line })}>
                    {line.map((token, key) => (
                      <span key={key} {...getTokenProps({ token })} />
                    ))}
                  </div>
                ))}
              </>
            )}
          </Highlight>
        )}
        onFocus={onFocus}
        onBlur={onBlur}
        disabled={disabled}
        readOnly={readOnly}
        placeholder={placeholder}
        name={name}
        className={className}
        padding={0}
        style={{
          ...(theme.typography.code as CSSProperties),
          width: '100%',
          fontSize: 13,
          lineHeight: 1.5,
          // roughly the 8 rows the plain textarea showed before
          minHeight: 160,
          // the transparent textarea's caret inherits this color
          color: theme.palette.text.primary,
        }}
        {...rest}
      />
    </div>
  );
});
