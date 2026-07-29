export type CSRFToken = {
  field: string,
  token: string,
};

export type FileInputElement = HTMLInputElement & { files: FileList };
