// NOTE(ben): Types for data returned by "API" endpoints, i.e. stuff called
// from the frontend.

export type CheckUsernameResult = { found: false } | {
  found: true,
  username: string,
  name: string,
  avatarUrl: string,
};
