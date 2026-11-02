// NOTE(ben): Types for data returned by "API" endpoints, i.e. stuff called
// from the frontend.

import { Project } from "./models";

export type BetaWeekSignupPayload = {
  projectID: number,
  doTest: boolean,
};

export type BetaWeekSignupResult = {
  projects: number[],
};

export type CheckUsernameResult = { found: false } | {
  found: true,
  username: string,
  name: string,
  avatarUrl: string,
};

export type ProjectResult = {
  project: Project,
  card: string,
};
