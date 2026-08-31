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
