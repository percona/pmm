import { renderMarkdown, textUtil } from '@grafana/data';
import highlight from 'highlight.js';

const highlightCode = (code: HTMLElement) => {
  const language = /language-([\w-]+)/.exec(code.className)?.[1];
  const raw = (code.textContent ?? '').replace(/\n$/, '');

  try {
    // highlight.js escapes the text it returns.
    // eslint-disable-next-line no-param-reassign
    code.innerHTML = language && highlight.getLanguage(language)
      ? highlight.highlight(raw, { language }).value
      : highlight.highlightAuto(raw).value;
    code.classList.add('hljs');
  } catch {
    // Unhighlighted code is still readable.
  }

  return language;
};

const FENCE = /^\s*```/;
const LIST_ITEM = /^\s*(\d+\.|[-*+])\s/;

// Only a list starting at 1 may interrupt a paragraph, so a model's "3." right under a paragraph line
// would be folded into it; a blank line keeps it an item.
const separateNumberedItems = (md: string): string => {
  let inFence = false;

  return md.split('\n').flatMap((line, i, lines) => {
    const prev = lines[i - 1] ?? '';
    const split = !inFence && /^\d+\.\s/.test(line) && prev.trim() !== ''
      && !FENCE.test(prev) && !LIST_ITEM.test(prev);

    if (FENCE.test(line)) inFence = !inFence;

    return split ? ['', line] : [line];
  }).join('\n');
};

/** Wraps a fenced block with its language and a copy button. */
const decorateCodeBlock = (pre: HTMLPreElement) => {
  const code = pre.querySelector('code');
  const language = code ? highlightCode(code) : undefined;
  const block = document.createElement('div');
  const header = document.createElement('div');
  const label = document.createElement('span');
  const copy = document.createElement('button');

  block.className = 'ai-code';
  header.className = 'ai-code-header';
  label.className = 'ai-code-lang';
  label.textContent = language ?? 'code';
  copy.type = 'button';
  copy.textContent = 'Copy';
  copy.setAttribute('data-copy-code', '');
  copy.setAttribute('aria-label', 'Copy code');
  header.append(label, copy);
  pre.replaceWith(block);
  block.append(header, pre);
};

/** Turns an AI Insights answer into HTML laid out like an Assistant answer. */
export const renderAnalysis = (md: string): string => {
  const template = document.createElement('template');

  // DOMPurify, unlike the text panel sanitizer, keeps <ol start>.
  template.innerHTML = textUtil.sanitize(renderMarkdown(separateNumberedItems(md), { noSanitize: true }));

  const root = template.content;

  // The answer is model output: keep its structure, not its styling.
  root.querySelectorAll('[style]').forEach((el) => el.removeAttribute('style'));
  // Each item carries its number, so lists split by code blocks keep counting from their start.
  root.querySelectorAll('ol').forEach((ol) => {
    Array.from(ol.children)
      .filter((li) => li.tagName === 'LI')
      .forEach((li, i) => li.setAttribute('data-n', String(ol.start + i)));
  });
  root.querySelectorAll('pre').forEach(decorateCodeBlock);
  root.querySelectorAll('table').forEach((table) => {
    const wrapper = document.createElement('div');

    wrapper.className = 'ai-table';
    table.replaceWith(wrapper);
    wrapper.append(table);
  });

  const out = document.createElement('div');

  out.append(root);

  return out.innerHTML;
};
