export interface DashboardFolder {
  id: number;
  uid: string;
  title: string;
}
export type GetFoldersResponse = DashboardFolder[];

export interface CreateFolderPayload {
  title: string;
}
