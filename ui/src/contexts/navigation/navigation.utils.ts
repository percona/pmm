import { DashboardFolder } from 'types/folder.types';
import { NAV_FOLDER_MAP } from './navigation.contants';
import { MenuItem } from './navigation.context.types';

export const addFolderLinks = (
  navTree: MenuItem[],
  folders: DashboardFolder[]
) => {
  for (const rootNode of navTree) {
    const id = rootNode.id + '-other-dashboards';
    const folder = folders.find(
      (f) => rootNode.id && NAV_FOLDER_MAP[rootNode.id] === f.title
    );
    const exists = rootNode.children?.some((i) => i.id === id);

    if (folder && !exists) {
      rootNode.children?.push({
        id,
        icon: 'search',
        title: 'Other dashboards',
        to: `/graph/dashboards/f/${folder.uid}/${rootNode.id}`,
      });
    }
  }
};
