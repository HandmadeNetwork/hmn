import { ProjectResult } from "./lib/apitypes";
import { must } from "./lib/utils";

const projectUrlField = must(document.querySelector<HTMLInputElement>("#project-url"));
const projectIDField = must(document.querySelector<HTMLInputElement>("#project-id"));
const projectPreviewContainer = must(document.querySelector<HTMLElement>("#project-preview"));

export type Config = {
  currentUserID: number,
  fetchProjectUrl: string,
};

export function init({
  currentUserID,
  fetchProjectUrl,
}: Config) {
  function clearProject() {
    projectPreviewContainer.hidden = true;
    projectPreviewContainer.innerHTML = "";
  }
  async function loadProject() {
    const projectUrl = projectUrlField.value.trim();
    if (!projectUrl) {
      return;
    }

    const res = await fetch(fetchProjectUrl, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        url: projectUrl,
      }),
    });
    if (res.status == 404) {
      projectPreviewContainer.hidden = false;
      projectPreviewContainer.innerText = "No project found."
      return;
    } else if (res.status >= 400) {
      console.error(res);
      throw new Error("bad request");
    }

    const project: ProjectResult = await res.json();
    if (!project.project.owners?.find(u => u.id === currentUserID)) {
      projectPreviewContainer.hidden = false;
      projectPreviewContainer.innerText = "You are not one of the owners of that project.";
      return;
    }

    projectIDField.value = `${project.project.id}`;
    projectPreviewContainer.hidden = false;
    projectPreviewContainer.innerHTML = project.card;
    projectPreviewContainer.querySelector("a")!.target = "_blank";
  };

  projectUrlField.addEventListener("input", clearProject);
  projectUrlField.addEventListener("change", loadProject);
  loadProject();
}
