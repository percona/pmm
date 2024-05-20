import { getCurrentVersion } from 'api/version';

export const getVersion = async () => {
  try {
    const res = await getCurrentVersion();
    return res;
  } catch (error) {
    const res = await getCurrentVersion({
      force: false,
      onlyInstalledVersion: true,
    });
    return res;
  }
};
