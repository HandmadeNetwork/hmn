import { BetaWeekSignupPayload, BetaWeekSignupResult } from "./lib/apitypes";
import { CSRFToken } from "./lib/types";

const buttons = Array.from(document.querySelectorAll<HTMLButtonElement>(".signup-btn"));

export type Config = {
  signupUrl: string,
  csrf: CSRFToken,
  currentSignups: number[],
  maxSignups: number,
}

export function init({
  signupUrl,
  csrf,
  currentSignups,
  maxSignups,
}: Config) {
  function updateButtons() {
    for (const button of buttons) {
      // NOTE(ben): Keep this in sync with the Go template.
      const projectID = parseInt(button.dataset.projectid!, 10);
      const isSignedUp = currentSignups.includes(projectID);
      const disabled = !isSignedUp && currentSignups.length >= maxSignups;
      button.innerText = isSignedUp
        ? "Remove signup"
        : "Sign up to test";
      button.disabled = disabled;
      button.classList.toggle("btn-primary", !disabled);
    }
  }

  async function signup(projectID: number) {
    try {
      for (const button of buttons) {
        button.disabled = true;
      }

      const res = await fetch(signupUrl, {
        method: "POST",
        headers: {
          "X-HMN-CSRF": csrf.token,
        },
        body: JSON.stringify({
          projectID,
          doTest: !currentSignups.includes(projectID),
        } satisfies BetaWeekSignupPayload),
      });
      if (!res.ok) {
        throw new Error("signup failed");
      }
      const body = await res.json() as BetaWeekSignupResult;
      currentSignups = body.projects;
    } finally {
      updateButtons();
    }
  }

  for (const button of buttons) {
    button.addEventListener("click", e => {
      e.preventDefault();
      const projectID = parseInt(button.dataset.projectid!, 10);
      signup(projectID);
    });
  }
  updateButtons();
}
