// src/rawdata/js/lib/utils.ts
function assert(cond, msg, soft = false) {
  if (!cond) {
    if (soft) {
      console.error(msg != null ? msg : "Assertion failed");
    } else {
      throw new Error(msg != null ? msg : "Assertion failed");
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
    assert(el, "no element with id ".concat(id));
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
      throw new Error("Couldn't find template with ID '".concat(id, "'"));
    }
    var root = templateEl.content.cloneNode(true);
    var paths = getTemplatePaths(id, root);
    var result = collectElements(paths, root);
    return result;
  };
}
function emptyElement(el) {
  var newEl = el.cloneNode(false);
  assert(el.parentElement);
  el.parentElement.insertBefore(newEl, el);
  el.parentElement.removeChild(el);
  return newEl;
}

// src/rawdata/js/snippetedit.ts
var snippetEditTemplate = makeTemplateCloner("snippet-edit");
var snippetEditProjectTemplate = makeTemplateCloner("snippet-edit-project");
function readableByteSize(numBytes) {
  const scales = [
    " bytes",
    "kb",
    "mb",
    "gb"
  ];
  let scale = 0;
  while (numBytes > 1024 && scale < scales.length - 1) {
    numBytes /= 1024;
    scale++;
  }
  return new Intl.NumberFormat([], { maximumFractionDigits: scale > 0 ? 2 : 0 }).format(numBytes) + scales[scale];
}
function makeSnippetEdit({
  maxFilesize,
  availableProjects,
  ownerName,
  ownerAvatar,
  ownerUrl,
  date,
  text,
  attachmentElement,
  projectIds,
  stickyProjectId,
  onDeleteRedirectUrl,
  snippetId,
  originalSnippetEl
}) {
  const snippetEdit = snippetEditTemplate();
  let projectSelector = null;
  let originalAttachment = null;
  const originalText = text;
  let attachmentChanged = false;
  let hasAttachment = false;
  snippetEdit.redirect.value = location.href;
  if (ownerAvatar) {
    assert(ownerUrl);
    snippetEdit.avatarImg.src = ownerAvatar;
    snippetEdit.avatarLink.href = ownerUrl;
    snippetEdit.avatarImg.hidden = false;
  } else {
    snippetEdit.avatarImg.hidden = true;
  }
  snippetEdit.username.textContent = ownerName != null ? ownerName : "";
  snippetEdit.username.href = ownerUrl != null ? ownerUrl : "";
  snippetEdit.date.textContent = new Intl.DateTimeFormat([], { month: "2-digit", day: "2-digit", year: "numeric" }).format(date);
  snippetEdit.text.value = text;
  if (attachmentElement) {
    originalAttachment = attachmentElement.cloneNode(true);
    clearAttachment(true);
  }
  if (snippetId !== void 0 && snippetId !== null) {
    snippetEdit.snippetId.value = snippetId;
  } else {
    snippetEdit.deleteButton.remove();
  }
  for (let i = 0; i < projectIds.length; ++i) {
    let proj = null;
    for (let j = 0; j < availableProjects.length; ++j) {
      if (projectIds[i] == availableProjects[j].id) {
        proj = availableProjects[j];
        break;
      }
    }
    if (proj) {
      addProject(proj);
    }
  }
  updateProjectSelector();
  if (originalSnippetEl) {
    snippetEdit.cancelLink.addEventListener("click", function() {
      cancel();
    });
  } else {
    snippetEdit.cancelLink.remove();
  }
  function cancel() {
    if (originalSnippetEl) {
      snippetEdit.root.parentElement.insertBefore(originalSnippetEl, snippetEdit.root);
    }
    snippetEdit.root.remove();
  }
  function addProject(proj) {
    let projEl = snippetEditProjectTemplate();
    projEl.projectId.value = "".concat(proj.id);
    projEl.projectLogo.src = proj.logo;
    projEl.projectName.textContent = proj.name;
    if (proj.id == stickyProjectId) {
      projEl.removeButton.remove();
    } else {
      projEl.removeButton.addEventListener("click", function(ev) {
        projEl.root.remove();
        updateProjectSelector();
      });
    }
    snippetEdit.projectList.appendChild(projEl.root);
  }
  function updateProjectSelector() {
    if (projectSelector) {
      projectSelector.remove();
    }
    let remainingProjects = [];
    let projInputs = snippetEdit.projectList.querySelectorAll("input[name=project_id]");
    let assignedIds = [];
    for (let i = 0; i < projInputs.length; ++i) {
      let id = parseInt(projInputs[i].value, 10);
      if (!isNaN(id)) {
        assignedIds.push(id);
      }
    }
    for (let i = 0; i < availableProjects.length; ++i) {
      let found = false;
      for (let j = 0; j < assignedIds.length; ++j) {
        if (assignedIds[j] == availableProjects[i].id) {
          found = true;
          break;
        }
      }
      if (!found) {
        remainingProjects.push(availableProjects[i]);
      }
    }
    if (remainingProjects.length > 0) {
      projectSelector = document.createElement("select");
      const option = document.createElement("option");
      option.textContent = "Add to project...";
      option.selected = true;
      projectSelector.appendChild(option);
      for (let i = 0; i < remainingProjects.length; ++i) {
        const option2 = document.createElement("option");
        option2.value = "".concat(remainingProjects[i].id);
        option2.selected = false;
        option2.textContent = remainingProjects[i].name;
        projectSelector.appendChild(option2);
      }
      projectSelector.addEventListener("change", (ev) => {
        assert(projectSelector);
        if (projectSelector.selectedOptions.length > 0) {
          let selected = projectSelector.selectedOptions[0];
          if (selected.value != "") {
            let id = parseInt(selected.value, 10);
            if (!isNaN(id)) {
              for (let i = 0; i < availableProjects.length; ++i) {
                if (availableProjects[i].id == id) {
                  addProject(availableProjects[i]);
                  break;
                }
              }
            }
            updateProjectSelector();
          }
        }
      });
      snippetEdit.projectList.appendChild(projectSelector);
    }
  }
  function setFile(file) {
    let dt = new DataTransfer();
    dt.items.add(file);
    snippetEdit.file.files = dt.files;
    attachmentChanged = true;
    snippetEdit.removeAttachment.value = "false";
    hasAttachment = true;
    let el = null;
    if (file.type.startsWith("image/")) {
      el = document.createElement("img");
      el.src = URL.createObjectURL(file);
    } else if (file.type.startsWith("video/")) {
      el = document.createElement("video");
      el.src = URL.createObjectURL(file);
      el.controls = true;
    } else if (file.type.startsWith("audio/")) {
      el = document.createElement("audio");
      el.src = URL.createObjectURL(file);
    } else {
      el = document.createElement("div");
      el.classList.add("project-card", "br2", "pv1", "ph2");
      let anchor = document.createElement("a");
      anchor.href = URL.createObjectURL(file);
      anchor.setAttribute("target", "_blank");
      anchor.textContent = file.name + " (" + readableByteSize(file.size) + ")";
      el.appendChild(anchor);
    }
    setPreview(el);
    validate();
  }
  function clearAttachment(restoreOriginal) {
    snippetEdit.file.value = "";
    let el = null;
    attachmentChanged = false;
    hasAttachment = false;
    snippetEdit.removeAttachment.value = "false";
    if (originalAttachment) {
      if (restoreOriginal) {
        hasAttachment = true;
        el = originalAttachment;
      } else {
        attachmentChanged = true;
        snippetEdit.removeAttachment.value = "true";
      }
    }
    setPreview(el);
    validate();
  }
  function setPreview(el) {
    if (el) {
      snippetEdit.uploadBox.style.display = "none";
      snippetEdit.previewBox.style.display = "block";
      snippetEdit.uploadResetBox.style.display = "none";
      snippetEdit.previewContent = emptyElement(snippetEdit.previewContent);
      snippetEdit.previewContent.appendChild(el);
      snippetEdit.resetLink.style.display = !originalAttachment || el == originalAttachment ? "none" : "inline-block";
    } else {
      snippetEdit.uploadBox.style.display = "block";
      snippetEdit.previewBox.style.display = "none";
      if (originalAttachment) {
        snippetEdit.uploadResetBox.style.display = "block";
      }
    }
  }
  function validate() {
    let sizeGood = true;
    if (snippetEdit.file.files.length > 0 && snippetEdit.file.files[0].size > maxFilesize) {
      let readableSize = new Intl.NumberFormat([], { useGrouping: true }).format(maxFilesize);
      snippetEdit.errors.textContent = "File is too big! Max filesize is " + readableSize + " bytes.";
      sizeGood = false;
    } else {
      snippetEdit.errors.textContent = "";
    }
    let hasText = snippetEdit.text.value.trim().length > 0;
    if ((hasText || hasAttachment) && sizeGood) {
      snippetEdit.saveButton.disabled = false;
    } else {
      snippetEdit.saveButton.disabled = true;
    }
  }
  snippetEdit.uploadLink.addEventListener("click", () => {
    snippetEdit.file.click();
  });
  snippetEdit.removeLink.addEventListener("click", () => {
    clearAttachment(false);
  });
  snippetEdit.replaceLink.addEventListener("click", () => {
    snippetEdit.file.click();
  });
  snippetEdit.resetLink.addEventListener("click", () => {
    clearAttachment(true);
  });
  snippetEdit.uploadResetLink.addEventListener("click", () => {
    clearAttachment(true);
  });
  snippetEdit.file.addEventListener("change", () => {
    if (snippetEdit.file.files.length > 0) {
      setFile(snippetEdit.file.files[0]);
    }
  });
  snippetEdit.root.addEventListener("dragover", (ev) => {
    assert(ev.dataTransfer);
    let effect = "none";
    for (let i = 0; i < ev.dataTransfer.items.length; ++i) {
      if (ev.dataTransfer.items[i].kind.toLowerCase() == "file") {
        effect = "copy";
        break;
      }
    }
    ev.dataTransfer.dropEffect = effect;
    ev.preventDefault();
  });
  let enterCounter = 0;
  snippetEdit.root.addEventListener("dragenter", (ev) => {
    assert(ev.dataTransfer);
    enterCounter++;
    const droppable = Array.from(ev.dataTransfer.items).some(
      (item) => item.kind.toLowerCase() === "file"
    );
    if (droppable) {
      snippetEdit.root.classList.add("drop");
    }
  });
  snippetEdit.root.addEventListener("dragleave", (ev) => {
    enterCounter--;
    if (enterCounter == 0) {
      snippetEdit.root.classList.remove("drop");
    }
  });
  snippetEdit.root.addEventListener("drop", (ev) => {
    enterCounter = 0;
    snippetEdit.root.classList.remove("drop");
    if (ev.dataTransfer && ev.dataTransfer.files && ev.dataTransfer.files.length > 0) {
      setFile(ev.dataTransfer.files[0]);
    }
    ev.preventDefault();
  });
  snippetEdit.text.addEventListener("paste", (ev) => {
    var _a;
    assert(ev.clipboardData);
    const files = (_a = ev.clipboardData.files) != null ? _a : [];
    if (files.length > 0) {
      setFile(files[0]);
    }
  });
  snippetEdit.text.addEventListener("input", () => {
    validate();
  });
  snippetEdit.saveButton.addEventListener("click", (ev) => {
    let projectsChanged = false;
    let projInputs = snippetEdit.projectList.querySelectorAll("input[name=project_id]");
    let assignedIds = [];
    for (let i = 0; i < projInputs.length; ++i) {
      let id = parseInt(projInputs[i].value, 10);
      if (!isNaN(id)) {
        assignedIds.push(id);
      }
    }
    if (projectIds.length != assignedIds.length) {
      projectsChanged = true;
    } else {
      for (let i = 0; i < projectIds.length; ++i) {
        let found = false;
        for (let j = 0; j < assignedIds.length; ++j) {
          if (projectIds[i] == assignedIds[j]) {
            found = true;
          }
        }
        if (!found) {
          projectsChanged = true;
          break;
        }
      }
    }
    if (originalSnippetEl && (!attachmentChanged && originalText == snippetEdit.text.value.trim() && !projectsChanged)) {
      ev.preventDefault();
      cancel();
    }
  });
  snippetEdit.deleteButton.addEventListener("click", function(ev) {
    if (!window.confirm("Are you sure you want to delete this snippet?")) {
      ev.preventDefault();
      return;
    }
    snippetEdit.redirect.value = onDeleteRedirectUrl != null ? onDeleteRedirectUrl : "";
    snippetEdit.file.value = "";
  });
  validate();
  return snippetEdit;
}
function editTimelineSnippet(timelineItemEl, {
  maxFilesize,
  availableProjects,
  stickyProjectId,
  onDeleteRedirectUrl
}) {
  var _a, _b, _c, _d, _e;
  const ownerName = (_a = timelineItemEl.querySelector(".user")) == null ? void 0 : _a.textContent;
  const ownerUrl = (_b = timelineItemEl.querySelector(".user")) == null ? void 0 : _b.href;
  const ownerAvatar = (_c = timelineItemEl.querySelector(".avatar")) == null ? void 0 : _c.src;
  const creationDate = new Date(must(timelineItemEl.querySelector("time")).dateTime);
  const rawDesc = must(timelineItemEl.querySelector(".rawdesc")).textContent;
  const attachment = (_e = (_d = timelineItemEl.querySelector(".timeline-media")) == null ? void 0 : _d.children) == null ? void 0 : _e[0];
  const projectIds = [];
  const projectEls = timelineItemEl.querySelectorAll(".project-id-list > input");
  for (let i = 0; i < projectEls.length; ++i) {
    let projid = parseInt(projectEls[i].value, 10);
    if (projid) {
      projectIds.push(projid);
    }
  }
  let snippetEdit = makeSnippetEdit({
    maxFilesize,
    availableProjects,
    ownerName,
    ownerAvatar,
    ownerUrl,
    date: creationDate,
    text: rawDesc,
    attachmentElement: attachment,
    projectIds,
    stickyProjectId,
    onDeleteRedirectUrl,
    snippetId: must(timelineItemEl.getAttribute("data-id")),
    originalSnippetEl: timelineItemEl
  });
  timelineItemEl.parentElement.insertBefore(snippetEdit.root, timelineItemEl);
  timelineItemEl.remove();
}
export {
  editTimelineSnippet,
  makeSnippetEdit
};
