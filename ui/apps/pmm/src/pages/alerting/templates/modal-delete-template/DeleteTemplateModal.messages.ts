export const Messages = {
  title: 'Delete alert template',
  content: (name: string) =>
    `Are you sure you want to delete the template "${name}"? This action cannot be undone.`,
  cancel: 'Cancel',
  confirm: 'Delete',
  success: 'Alert template deleted successfully',
  error: 'Failed to delete alert template',
};
