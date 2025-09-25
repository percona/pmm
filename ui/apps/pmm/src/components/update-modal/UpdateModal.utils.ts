export const parseReleaseHighlights = (releaseNotes?: string) => {
  if (!releaseNotes) {
    return '';
  }

  const startIdx =
    releaseNotes.indexOf('Release summary') ||
    releaseNotes.indexOf('Release Summary');
  const endIdx = releaseNotes.indexOf('##', startIdx + 15);

  const highlights = releaseNotes.slice(startIdx + 15, endIdx);

  return highlights.trim();
};
