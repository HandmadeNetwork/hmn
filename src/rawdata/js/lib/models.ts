// NOTE(ben): Types corresponding to data from templates/types.go.

export type Asset = {
  url: string,

  id: string,
  filename: string,
  size: number,
  mimeType: string,
  width: number,
  height: number,
};

export type Icon = {
  name: string,
  svg: string,
};

export type SnippetEditorConfig = {
  assetMaxSize: number,
  availableProjects: SnippetEditAvailableProject[],
  owner: User,
  requiredProjectID?: number,

  submitUrl: string,
  onDeleteRedirectUrl?: string,
};

export type SnippetEditAvailableProject = {
  id: number,
  name: string,
  logo: string,
};

export type User = {
  id: number,
  username: string,
  name: string,
  avatar: Asset | null,
  avatarUrl?: string,
  profileUrl: string,
};
