import { initReorderable } from "./reorderable";
import { makeTemplateCloner } from "./templates";
import { must } from "./utils";

export type LinkData = Array<{
  name: string,
  url: string,
  primary: boolean,
}>;

type LinkEditorRow = {
  rootElement: HTMLElement,
  nameInput: HTMLInputElement,
  urlInput: HTMLInputElement,
  grabHandle: HTMLElement,
  deleteButton: HTMLAnchorElement,
};

type EmptySectionTemplate = {
  rootElement: HTMLElement,
};

const linksContainer = must(document.querySelector("#links"));
const parentForm = must(linksContainer.closest("form"));
const addButton = must(document.querySelector("#link-editor-add-button"));
const primaryLinksTitle = must(linksContainer.querySelector(".primary-links"));
const secondaryLinksTitle = must(linksContainer.querySelector(".secondary-links"));
const linksJSONInput = must(document.querySelector("#links-json")) as HTMLInputElement;

const linkTemplate = makeTemplateCloner<LinkEditorRow>("link-editor-row");
const emptySectionTemplate = makeTemplateCloner<EmptySectionTemplate>("link-editor-empty-section");

const { startDrag: startLinkDrag } = initReorderable(
  linksContainer,
  {
    onReorder(item) {
      ensurePlaceholders();
      linksUpdated();
    },
  },
);

function makeLink(): LinkEditorRow {
  const res = linkTemplate();
  res.nameInput.addEventListener("input", linkInput);
  res.urlInput.addEventListener("input", linkInput);
  res.grabHandle.addEventListener("pointerdown", startLinkDrag);
  res.deleteButton.addEventListener("click", e => {
    e.preventDefault();
    const link = must((e.target as typeof res.deleteButton).closest(".link-editor-row"));
    deleteLink(link);
  });
  return res;
}

function ensurePlaceholders() {
  for (const el of linksContainer.querySelectorAll(".link-placeholder")) {
    el.remove();
  }

  let numPrimary = 0, numSecondary = 0;
  let primary = true;
  for (const el of linksContainer.children) {
    if (el === primaryLinksTitle) {
      continue;
    }
    if (el === secondaryLinksTitle) {
      primary = false;
      continue;
    }

    if (el.classList.contains("reorderable-item")) {
      if (primary) {
        numPrimary += 1;
      } else {
        numSecondary += 1;
      }
    }
  }

  if (numPrimary === 0) {
    primaryLinksTitle.insertAdjacentElement("afterend", emptySectionTemplate().rootElement);
  }
  if (numSecondary === 0) {
    secondaryLinksTitle.insertAdjacentElement("afterend", emptySectionTemplate().rootElement);
  }
}
ensurePlaceholders();

export function getLinkData(): LinkData {
  const links = [];
  let primary = true;
  for (const el of linksContainer.children) {
    if (el === secondaryLinksTitle) {
      primary = false;
      continue;
    }

    if (el.classList.contains("link-editor-row")) {
      const name = (must(el.querySelector(".link-name")) as HTMLInputElement).value;
      const url = (must(el.querySelector(".link-url")) as HTMLInputElement).value;
      if (!url) {
        continue;
      }
      links.push({ name, url, primary });
    }
  }
  return links;
}
function updateLinksJSON() {
  linksJSONInput.value = JSON.stringify(getLinkData());
}
function linksUpdated() {
  updateLinksJSON();
  window.dispatchEvent(new Event("linkedit"));
}

function addLink() {
  secondaryLinksTitle.insertAdjacentElement("beforebegin", makeLink().rootElement);

  ensurePlaceholders();
  linksUpdated();
}
function deleteLink(row: Element) {
  row.remove();

  ensurePlaceholders();
  linksUpdated();
}
function linkInput() {
  linksUpdated();
}

export function initLinkEditor(initialLinks: LinkData) {
  for (const link of initialLinks) {
    const l = makeLink();
    l.nameInput.value = link.name;
    l.urlInput.value = link.url;
    if (link.primary) {
      secondaryLinksTitle.insertAdjacentElement("beforebegin", l.rootElement);
    } else {
      linksContainer.appendChild(l.rootElement);
    }
  }

  addButton.addEventListener("click", e => {
    e.preventDefault();
    addLink();
  })

  parentForm.addEventListener("submit", function () {
    updateLinksJSON(); // NOTE(ben): Just update JSON, don't fire another event
  });
}
