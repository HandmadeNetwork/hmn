// src/rawdata/js/betaweek_2026_index.ts
var buttons = Array.from(document.querySelectorAll(".signup-btn"));
function init({
  signupUrl,
  csrf,
  currentSignups,
  maxSignups
}) {
  function updateButtons() {
    for (const button of buttons) {
      const projectID = parseInt(button.dataset.projectid, 10);
      const isSignedUp = currentSignups.includes(projectID);
      const disabled = !isSignedUp && currentSignups.length >= maxSignups;
      button.innerText = isSignedUp ? "Remove signup" : "Sign up to test";
      button.disabled = disabled;
      button.classList.toggle("btn-primary", !disabled);
    }
  }
  async function signup(projectID) {
    try {
      for (const button of buttons) {
        button.disabled = true;
      }
      const res = await fetch(signupUrl, {
        method: "POST",
        headers: {
          "X-HMN-CSRF": csrf.token
        },
        body: JSON.stringify({
          projectID,
          doTest: !currentSignups.includes(projectID)
        })
      });
      if (!res.ok) {
        throw new Error("signup failed");
      }
      const body = await res.json();
      currentSignups = body.projects;
    } finally {
      updateButtons();
    }
  }
  for (const button of buttons) {
    button.addEventListener("click", (e) => {
      e.preventDefault();
      const projectID = parseInt(button.dataset.projectid, 10);
      signup(projectID);
    });
  }
  updateButtons();
}
export {
  init
};
//# sourceMappingURL=betaweek_2026_index.js.map
