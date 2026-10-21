// A command a step asks to run; it joins the step above it as code.
const SQL_RE =
  /^(SELECT|SET|SHOW|ALTER|CREATE|DROP|DELETE|INSERT|UPDATE|START|STOP|RESET|FLUSH|KILL|GRANT|REVOKE|CALL|EXPLAIN|ANALYZE|VACUUM|REINDEX|CHANGE|PURGE|OPTIMIZE|BEGIN|COMMIT)\b/i;
const SHELL_RE =
  /^(pt-|mysql|mysqladmin|mysqldump|psql|pg_|mongosh|mongo\b|db\.|rs\.|sh\.|sudo\b|systemctl\b|kubectl\b|docker\b|pmm-admin\b|curl\b)/;

const commandLanguage = (step: string) => {
  if (SQL_RE.test(step)) return 'sql';
  if (SHELL_RE.test(step)) return 'bash';
  return undefined;
};

type Part = { note: string } | { lang: string; lines: string[] };

/** Numbered markdown for remediation steps stored one line each: commands become code under their step. */
export const stepsMarkdown = (steps: string[]): string => {
  const items: { text: string; parts: Part[] }[] = [];
  const current = () => {
    if (!items.length) items.push({ text: '', parts: [] });
    return items[items.length - 1];
  };
  steps
    .map((s) => s.trim())
    // Older investigations stored fence markers as steps of their own.
    .filter((s) => s !== '' && !s.startsWith('```'))
    .forEach((step) => {
      const lang = commandLanguage(step);
      if (lang) {
        const parts = current().parts;
        const last = parts[parts.length - 1];
        if (last && 'lang' in last && last.lang === lang) last.lines.push(step);
        else parts.push({ lang, lines: [step] });
      } else if (/^[-*] /.test(step) && items.length) {
        current().parts.push({ note: step.slice(2) });
      } else {
        items.push({ text: step, parts: [] });
      }
    });

  return items
    .map((item, i) => {
      const marker = `${i + 1}. `;
      const pad = ' '.repeat(marker.length);
      const blocks = item.parts.map((p) =>
        'note' in p
          ? pad + p.note
          : [`\`\`\`${p.lang}`, ...p.lines, '```']
              .map((l) => pad + l)
              .join('\n')
      );
      return [marker + item.text, ...blocks].join('\n\n');
    })
    .join('\n\n');
};
