export const copyToClipboard = async (text: string): Promise<boolean> => {
  if (!navigator.clipboard || !window.isSecureContext) {
    return false;
  }
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
};
