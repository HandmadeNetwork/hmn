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

// src/rawdata/js/lib/templates.ts
var templateElementCache = {};
var templatePathCache = {};
function getTemplateEl(id) {
  if (!templateElementCache[id]) {
    const el = document.getElementById(id);
    assert(el, `no element with id ${id}`);
    assert(el instanceof HTMLTemplateElement);
    templateElementCache[id] = el;
  }
  return templateElementCache[id];
}
function getTemplatePaths(id, rootNode) {
  if (!templatePathCache[id]) {
    let descend2 = function(path, el) {
      for (var i = 0; i < el.children.length; ++i) {
        var child = el.children[i];
        var childPath = path.concat([i]);
        var tmplName = child.getAttribute("data-tmpl");
        if (tmplName) {
          paths.push([tmplName, childPath]);
        }
        if (child.children.length > 0) {
          descend2(childPath, child);
        }
      }
    };
    var descend = descend2;
    var paths = [];
    paths.push(["root", []]);
    descend2([], rootNode);
    templatePathCache[id] = paths;
  }
  return templatePathCache[id];
}
function collectElements(paths, rootElement) {
  var result = {};
  for (var i = 0; i < paths.length; ++i) {
    var path = paths[i];
    var current = rootElement;
    for (var j = 0; j < path[1].length; ++j) {
      current = current.children[path[1][j]];
    }
    result[path[0]] = current;
  }
  return result;
}
function makeTemplateCloner(id) {
  return function() {
    var templateEl = getTemplateEl(id);
    if (templateEl === null) {
      throw new Error(`Couldn't find template with ID '${id}'`);
    }
    var root = templateEl.content.cloneNode(true);
    var paths = getTemplatePaths(id, root);
    var result = collectElements(paths, root);
    return result;
  };
}

// src/rawdata/js/lib/image_selector.ts
var imageSelectorTemplate = makeTemplateCloner("image-selector");
var ImageSelector = class {
  // The current URL of the selector. (Will be an object URL if the image has
  // not been submitted yet.)
  url;
  // The template content to be inserted into the DOM wherever you like.
  root;
  // 
  maxFileSize;
  imageInput;
  removeImageInput;
  previewImage;
  previewContainer;
  resetLink;
  removeLink;
  filenameText;
  errorEl;
  defaultUrl;
  originalUrl;
  originalFilename;
  onUpdate;
  constructor(formName, maxFileSize, {
    defaultUrl = "",
    originalUrl = "",
    originalFilename = "",
    onUpdate = () => {
    }
  } = {}) {
    const tmpl = imageSelectorTemplate();
    this.url = "";
    this.root = tmpl.root;
    this.maxFileSize = maxFileSize;
    this.imageInput = tmpl.inputImage;
    this.removeImageInput = tmpl.inputRemove;
    this.previewImage = tmpl.preview;
    this.previewContainer = tmpl.previewContainer;
    this.resetLink = tmpl.linkReset;
    this.removeLink = tmpl.linkRemove;
    this.filenameText = tmpl.filenameText;
    this.errorEl = tmpl.errorMessage;
    this.defaultUrl = defaultUrl;
    this.originalUrl = originalUrl;
    this.originalFilename = originalFilename;
    this.onUpdate = onUpdate;
    this.imageInput.name = formName;
    this.imageInput.value = "";
    this.removeImageInput.name = `remove_${formName}`;
    this.removeImageInput.value = "";
    this.setImageUrl(
      this.originalUrl,
      /*initial=*/
      true
    );
    this.updatePreview();
    this.imageInput.addEventListener("change", (ev) => {
      if (this.imageInput.files.length > 0) {
        this.handleNewImageFile(this.imageInput.files[0]);
      }
    });
    this.resetLink.addEventListener("click", (ev) => {
      this.resetImage();
    });
    if (this.removeLink) {
      this.removeLink.addEventListener("click", (ev) => {
        this.removeImage();
      });
    }
  }
  openImageInput() {
    this.imageInput.click();
  }
  setImageUrl(url, initial = false) {
    this.url = url;
    if (url) {
      this.previewImage.style.display = "block";
      this.previewImage.src = url;
    } else {
      this.previewImage.style.display = "none";
      this.previewImage.removeAttribute("src");
    }
    if (!initial) {
      this.onUpdate(url);
    }
  }
  setError(error) {
    this.errorEl.textContent = error;
    this.errorEl.hidden = !error;
    this.imageInput.setCustomValidity(error);
    this.imageInput.reportValidity();
  }
  checkSizeLimit(size) {
    if (size > this.maxFileSize) {
      this.setError("File too big. Max file size is " + this.maxFileSize + " bytes.");
    } else {
      this.setError("");
    }
  }
  updatePreview(file = null) {
    const showReset = this.originalUrl && this.originalUrl !== this.defaultUrl && this.originalUrl !== this.url;
    const showRemove = this.url !== this.defaultUrl;
    this.resetLink.hidden = !showReset;
    this.removeLink.hidden = !showRemove;
    if (this.url === this.originalUrl) {
      this.filenameText.innerText = this.originalFilename;
    } else {
      this.filenameText.innerText = file ? file.name : "";
    }
    this.previewContainer.hidden = !this.url;
  }
  handleNewImageFile(file) {
    this.checkSizeLimit(file.size);
    this.removeImageInput.value = "";
    this.setImageUrl(URL.createObjectURL(file));
    this.updatePreview(file);
  }
  removeImage() {
    this.checkSizeLimit(0);
    this.imageInput.value = "";
    this.removeImageInput.value = "true";
    this.setImageUrl(this.defaultUrl);
    this.updatePreview(null);
  }
  resetImage() {
    this.checkSizeLimit(0);
    this.imageInput.value = "";
    this.removeImageInput.value = "";
    this.setImageUrl(this.originalUrl);
    this.updatePreview(null);
  }
};

// src/rawdata/js/lib/reorderable.js
function initReorderable(container, {
  onReorder = (item) => {
  }
} = {}) {
  let dragItem = null;
  let dragPointerId = null;
  let dragItemStartY = 0;
  let dragMouseStartY = 0;
  const dummy = document.createElement("div");
  dummy.classList.add("reorderable-dummy");
  function startDrag(e) {
    if (!e.isPrimary || e.button !== 0) {
      return;
    }
    e.preventDefault();
    const item = e.target.closest(".reorderable-item");
    const top = item.offsetTop;
    item.style.position = "absolute";
    item.style.top = `${top}px`;
    item.classList.add("reorderable-dragging");
    dummy.style.height = `${item.offsetHeight}px`;
    item.insertAdjacentElement("beforebegin", dummy);
    document.body.classList.add("grabbing");
    dragItem = item;
    dragPointerId = e.pointerId;
    dragItemStartY = top;
    dragMouseStartY = e.pageY;
    container.setPointerCapture(e.pointerId);
    container.addEventListener("pointermove", doDrag);
    container.addEventListener("lostpointercapture", endDrag, { once: true });
  }
  function doDrag(e) {
    const delta = e.pageY - dragMouseStartY;
    const top = dragItemStartY + delta;
    const middle = top + dragItem.offsetHeight / 2;
    const items = container.querySelectorAll(".reorderable-item");
    let closestItem = null;
    let closestItemDist = Infinity;
    let insertBefore = null;
    for (const item of items) {
      if (item === dragItem) {
        continue;
      }
      const itemMiddle = item.offsetTop + item.offsetHeight / 2;
      const dist = middle - itemMiddle;
      if (Math.abs(dist) < closestItemDist) {
        closestItem = item;
        closestItemDist = Math.abs(dist);
        insertBefore = dist < 0;
      }
    }
    if (closestItem) {
      let alreadyOrdered = true;
      let n = closestItem;
      while (true) {
        if (insertBefore) {
          n = n.previousSibling;
        } else {
          n = n.nextSibling;
        }
        if (!n) {
          alreadyOrdered = false;
          break;
        }
        if (n === dragItem) {
          break;
        }
        if (n.classList?.contains("reorderable-item")) {
          alreadyOrdered = false;
          break;
        }
      }
      if (!alreadyOrdered) {
        closestItem.insertAdjacentElement(insertBefore ? "beforebegin" : "afterend", dummy);
        dragItem.remove();
        dummy.insertAdjacentElement("beforebegin", dragItem);
        onReorder(dragItem);
      }
    }
    const maxTop = container.offsetHeight - dragItem.offsetHeight;
    const newTop = Math.max(0, Math.min(maxTop, top));
    dragItem.style.top = `${newTop}px`;
  }
  function endDrag(e) {
    container.removeEventListener("pointermove", doDrag);
    dragItem.remove();
    dummy.insertAdjacentElement("beforebegin", dragItem);
    dummy.remove();
    dragItem.style.position = null;
    dragItem.style.top = null;
    dragItem.classList.remove("reorderable-dragging");
    document.body.classList.remove("grabbing");
    onReorder(dragItem);
    dragItem = null;
    dragPointerId = null;
    dragItemStartY = 0;
    dragMouseStartY = 0;
  }
  return {
    startDrag
  };
}

// src/rawdata/js/lib/link_editor.ts
var linksContainer = must(document.querySelector("#links"));
var parentForm = must(linksContainer.closest("form"));
var addButton = must(document.querySelector("#link-editor-add-button"));
var primaryLinksTitle = must(linksContainer.querySelector(".primary-links"));
var secondaryLinksTitle = must(linksContainer.querySelector(".secondary-links"));
var linksJSONInput = must(document.querySelector("#links-json"));
var linkTemplate = makeTemplateCloner("link-editor-row");
var emptySectionTemplate = makeTemplateCloner("link-editor-empty-section");
var { startDrag: startLinkDrag } = initReorderable(
  linksContainer,
  {
    onReorder(item) {
      ensurePlaceholders();
      linksUpdated();
    }
  }
);
function makeLink() {
  const res = linkTemplate();
  res.nameInput.addEventListener("input", linkInput);
  res.urlInput.addEventListener("input", linkInput);
  res.grabHandle.addEventListener("pointerdown", startLinkDrag);
  res.deleteButton.addEventListener("click", (e) => {
    e.preventDefault();
    const link = must(e.target.closest(".link-editor-row"));
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
function getLinkData() {
  const links = [];
  let primary = true;
  for (const el of linksContainer.children) {
    if (el === secondaryLinksTitle) {
      primary = false;
      continue;
    }
    if (el.classList.contains("link-editor-row")) {
      const name = must(el.querySelector(".link-name")).value;
      const url = must(el.querySelector(".link-url")).value;
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
function deleteLink(row) {
  row.remove();
  ensurePlaceholders();
  linksUpdated();
}
function linkInput() {
  linksUpdated();
}
function initLinkEditor(initialLinks) {
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
  ensurePlaceholders();
  addButton.addEventListener("click", (e) => {
    e.preventDefault();
    addLink();
  });
  parentForm.addEventListener("submit", function() {
    updateLinksJSON();
  });
}

// src/rawdata/js/lib/tabs.ts
function initTabs(container, {
  initialTab,
  onSelect = () => {
  }
} = {}) {
  const buttons = Array.from(container.querySelectorAll("[data-tab-button]"));
  const tabs = Array.from(container.querySelectorAll("[data-tab]"));
  const firstTab = tabs[0].getAttribute("data-tab");
  function selectTab(name, { sendEvent = true } = {}) {
    if (!container.querySelector(`[data-tab="${name}"]`)) {
      console.warn("no tab found with name", name);
      return selectTab(firstTab, { sendEvent });
    }
    for (const tab of tabs) {
      tab.hidden = tab.getAttribute("data-tab") !== name;
    }
    for (const button of buttons) {
      button.classList.toggle("tab-button-active", button.getAttribute("data-tab-button") === name);
    }
    if (sendEvent) {
      onSelect(name);
    }
  }
  selectTab(initialTab || firstTab, { sendEvent: false });
  for (const button of buttons) {
    button.addEventListener("click", () => {
      selectTab(button.getAttribute("data-tab-button"));
    });
  }
  return {
    selectTab
  };
}
function initHashTabs(container, {
  initialTab
} = {}) {
  const res = initTabs(container, {
    initialTab: initialTab ?? document.location.hash.substring(1),
    onSelect(name) {
      document.location.hash = `#${name}`;
    }
  });
  const { selectTab } = res;
  window.addEventListener("hashchange", (e) => {
    const tab = new URL(e.newURL).hash.substring(1);
    if (tab) {
      selectTab(tab, { sendEvent: false });
    }
  });
  return res;
}

// src/rawdata/js/user_settings.ts
function lengthReporter(inputEl, lengthEl) {
  let updateLength = function() {
    lengthEl.textContent = `${inputEl.value.length}/${inputEl.getAttribute("maxlength")}`;
  };
  inputEl.addEventListener("input", updateLength);
  updateLength();
}
function init({
  avatarMaxFileSize,
  avatarUrl,
  avatarFilename,
  linksJson
}) {
  lengthReporter(
    must(document.getElementById("realname")),
    must(document.querySelector(".realname-length"))
  );
  let avatarSelector = new ImageSelector(
    "user_avatar",
    avatarMaxFileSize,
    {
      originalUrl: avatarUrl,
      originalFilename: avatarFilename
    }
  );
  must(document.querySelector("#upload-avatar-button")).addEventListener("click", (e) => {
    e.preventDefault();
    avatarSelector.openImageInput();
  });
  must(document.querySelector("#user-avatar-placeholder")).replaceWith(avatarSelector.root);
  initLinkEditor(JSON.parse(linksJson));
  lengthReporter(
    must(document.getElementById("shortbio")),
    must(document.querySelector(".shortbio-length"))
  );
  lengthReporter(
    must(document.getElementById("longbio")),
    must(document.querySelector(".longbio-length"))
  );
  if (document.getElementById("signature")) {
    lengthReporter(
      must(document.getElementById("signature")),
      must(document.querySelector(".signature-length"))
    );
  }
  initHashTabs(document);
  const discordUnlinkForm = must(document.querySelector("#discord-unlink-form"));
  document.querySelector("#unlink-discord-button")?.addEventListener("click", (e) => {
    e.preventDefault();
    discordUnlinkForm.submit();
  });
  const discordShowcaseBacklogForm = must(document.querySelector("#discord-showcase-backlog"));
  document.querySelector("#discord-showcase-backlog-button")?.addEventListener("click", (e) => {
    e.preventDefault();
    discordShowcaseBacklogForm.submit();
  });
}
export {
  init
};
