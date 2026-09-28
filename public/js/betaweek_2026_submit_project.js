// src/rawdata/js/lib/utils.ts
function assert(cond, msg, soft = false) {
  if (!cond) {
    if (soft) {
      console.error(msg ?? "Assertion failed");
    } else {
      throw new Error(msg ?? "Assertion failed");
    }
  }
}
function must(val, msg) {
  assert(val, msg);
  return val;
}

// src/rawdata/js/betaweek_2026_submit_project.ts
var projectUrlField = must(document.querySelector("#project-url"));
var projectIDField = must(document.querySelector("#project-id"));
var projectPreviewContainer = must(document.querySelector("#project-preview"));
function init({
  currentUserID,
  fetchProjectUrl
}) {
  function clearProject() {
    projectPreviewContainer.hidden = true;
    projectPreviewContainer.innerHTML = "";
    projectIDField.value = "";
  }
  async function loadProject() {
    const projectUrl = projectUrlField.value.trim();
    if (!projectUrl) {
      return;
    }
    const res = await fetch(fetchProjectUrl, {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify({
        url: projectUrl
      })
    });
    if (res.status == 404) {
      projectPreviewContainer.hidden = false;
      projectPreviewContainer.innerText = "No project found.";
      return;
    } else if (res.status >= 400) {
      console.error(res);
      throw new Error("bad request");
    }
    const project = await res.json();
    if (!project.project.owners?.find((u) => u.id === currentUserID)) {
      projectPreviewContainer.hidden = false;
      projectPreviewContainer.innerText = "You are not one of the owners of that project.";
      return;
    }
    projectIDField.value = `${project.project.id}`;
    projectPreviewContainer.hidden = false;
    projectPreviewContainer.innerHTML = project.card;
    projectPreviewContainer.querySelector("a").target = "_blank";
  }
  ;
  projectUrlField.addEventListener("input", clearProject);
  projectUrlField.addEventListener("change", loadProject);
  loadProject();
}
export {
  init
};
//# sourceMappingURL=betaweek_2026_submit_project.js.map
