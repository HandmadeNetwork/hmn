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

export type Flowsnake = {
  angle: number,
  hue: number,
  size: number,
}

export type Project = {
  id: number,
  name: string,
  subdomain: string,
  color1: string,
  color2: string,
  url: string,
  blurb: string,
  description: string,
  ai_policy: string,
  owners: User[] | null,
  logo: string,
  flowsnake: Flowsnake,
  lifecycle: string,
  has_blog: boolean,
  has_forum: boolean,
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
