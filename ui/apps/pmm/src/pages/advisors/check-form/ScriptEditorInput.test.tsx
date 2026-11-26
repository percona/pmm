import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { ScriptEditorInput } from './ScriptEditorInput';

// Vitest unwraps CJS default exports, a Vite 8 production build does not:
// reproduce the build's binding of the default import to `module.exports`
vi.mock('react-simple-code-editor', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('react-simple-code-editor')>();
  return { default: { __esModule: true, default: actual.default } };
});

describe('ScriptEditorInput', () => {
  it('renders when the default import is the whole CJS module', () => {
    render(<ScriptEditorInput value="print(1)" onChange={vi.fn()} />);

    expect(screen.getByRole('textbox')).toHaveValue('print(1)');
  });
});
