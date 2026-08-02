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
  originalImageInput;
  newImageInput;
  removeImageInput;
  previewImage;
  previewContainer;
  resetLink;
  removeLink;
  filenameText;
  errorEl;
  defaultUrl;
  originalFile;
  onUpdate;
  onRemove;
  constructor(formName, maxFileSize, {
    defaultUrl = "",
    original,
    onUpdate = () => {
    },
    onRemove = () => {
    }
  } = {}) {
    const tmpl = imageSelectorTemplate();
    this.url = "";
    this.root = tmpl.root;
    this.maxFileSize = maxFileSize;
    this.originalImageInput = tmpl.inputOriginal;
    this.newImageInput = tmpl.inputImage;
    this.removeImageInput = tmpl.inputRemove;
    this.previewImage = tmpl.preview;
    this.previewContainer = tmpl.previewContainer;
    this.resetLink = tmpl.linkReset;
    this.removeLink = tmpl.linkRemove;
    this.filenameText = tmpl.filenameText;
    this.errorEl = tmpl.errorMessage;
    this.defaultUrl = defaultUrl;
    this.originalFile = original ?? null;
    this.onUpdate = onUpdate;
    this.onRemove = onRemove;
    this.originalImageInput.name = `original_${formName}`;
    this.originalImageInput.value = original?.id ?? "NOASSET";
    this.newImageInput.name = `image_${formName}`;
    this.newImageInput.value = "";
    this.removeImageInput.name = `remove_${formName}`;
    this.removeImageInput.value = "";
    this.setImageUrl(
      this.originalFile?.url ?? "",
      /*initial=*/
      true
    );
    this.updatePreview();
    this.newImageInput.addEventListener("change", (ev) => {
      if (this.newImageInput.files.length > 0) {
        this.handleNewImageFile(this.newImageInput.files[0]);
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
    return new Promise((resolve) => {
      const done = () => {
        if (this.newImageInput.files.length > 0) {
          resolve(this.newImageInput.files[0]);
        } else {
          resolve(null);
        }
      };
      this.newImageInput.addEventListener("change", done, { once: true });
      this.newImageInput.addEventListener("cancel", done, { once: true });
      this.newImageInput.click();
    });
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
    this.newImageInput.setCustomValidity(error);
    this.newImageInput.reportValidity();
  }
  checkSizeLimit(size) {
    if (size > this.maxFileSize) {
      this.setError("File too big. Max file size is " + this.maxFileSize + " bytes.");
    } else {
      this.setError("");
    }
  }
  updatePreview(file = null) {
    const showReset = this.originalFile && this.originalFile.url !== this.defaultUrl && this.originalFile.url !== this.url;
    const showRemove = this.url !== this.defaultUrl;
    this.resetLink.hidden = !showReset;
    this.removeLink.hidden = !showRemove;
    if (this.originalFile && this.url === this.originalFile.url) {
      this.filenameText.innerText = this.originalFile.filename;
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
    this.newImageInput.value = "";
    this.removeImageInput.value = this.originalFile?.id ?? "";
    this.setImageUrl(this.defaultUrl);
    this.updatePreview(null);
    this.onRemove();
  }
  resetImage() {
    this.checkSizeLimit(0);
    this.newImageInput.value = "";
    this.removeImageInput.value = "";
    this.setImageUrl(this.originalFile?.url ?? "");
    this.updatePreview(null);
  }
};

// src/rawdata/js/lib/base64.ts
function uint6ToB64(nUint6) {
  return nUint6 < 26 ? nUint6 + 65 : nUint6 < 52 ? nUint6 + 71 : nUint6 < 62 ? nUint6 - 4 : nUint6 === 62 ? 43 : nUint6 === 63 ? 47 : 65;
}
function base64EncArr(aBytes) {
  var nMod3 = 2, sB64Enc = "";
  for (var nLen = aBytes.length, nUint24 = 0, nIdx = 0; nIdx < nLen; nIdx++) {
    nMod3 = nIdx % 3;
    if (nIdx > 0 && nIdx * 4 / 3 % 76 === 0) {
      sB64Enc += "\r\n";
    }
    nUint24 |= aBytes[nIdx] << (16 >>> nMod3 & 24);
    if (nMod3 === 2 || aBytes.length - nIdx === 1) {
      sB64Enc += String.fromCharCode(uint6ToB64(nUint24 >>> 18 & 63), uint6ToB64(nUint24 >>> 12 & 63), uint6ToB64(nUint24 >>> 6 & 63), uint6ToB64(nUint24 & 63));
      nUint24 = 0;
    }
  }
  return sB64Enc.substr(0, sB64Enc.length - 2 + nMod3) + (nMod3 === 2 ? "" : nMod3 === 1 ? "=" : "==");
}
function strToUTF8Arr(sDOMStr) {
  var aBytes, nChr, nStrLen = sDOMStr.length, nArrLen = 0;
  for (var nMapIdx = 0; nMapIdx < nStrLen; nMapIdx++) {
    nChr = sDOMStr.charCodeAt(nMapIdx);
    nArrLen += nChr < 128 ? 1 : nChr < 2048 ? 2 : nChr < 65536 ? 3 : nChr < 2097152 ? 4 : nChr < 67108864 ? 5 : 6;
  }
  aBytes = new Uint8Array(nArrLen);
  for (var nIdx = 0, nChrIdx = 0; nIdx < nArrLen; nChrIdx++) {
    nChr = sDOMStr.charCodeAt(nChrIdx);
    if (nChr < 128) {
      aBytes[nIdx++] = nChr;
    } else if (nChr < 2048) {
      aBytes[nIdx++] = 192 + (nChr >>> 6);
      aBytes[nIdx++] = 128 + (nChr & 63);
    } else if (nChr < 65536) {
      aBytes[nIdx++] = 224 + (nChr >>> 12);
      aBytes[nIdx++] = 128 + (nChr >>> 6 & 63);
      aBytes[nIdx++] = 128 + (nChr & 63);
    } else if (nChr < 2097152) {
      aBytes[nIdx++] = 240 + (nChr >>> 18);
      aBytes[nIdx++] = 128 + (nChr >>> 12 & 63);
      aBytes[nIdx++] = 128 + (nChr >>> 6 & 63);
      aBytes[nIdx++] = 128 + (nChr & 63);
    } else if (nChr < 67108864) {
      aBytes[nIdx++] = 248 + (nChr >>> 24);
      aBytes[nIdx++] = 128 + (nChr >>> 18 & 63);
      aBytes[nIdx++] = 128 + (nChr >>> 12 & 63);
      aBytes[nIdx++] = 128 + (nChr >>> 6 & 63);
      aBytes[nIdx++] = 128 + (nChr & 63);
    } else {
      aBytes[nIdx++] = 252 + (nChr >>> 30);
      aBytes[nIdx++] = 128 + (nChr >>> 24 & 63);
      aBytes[nIdx++] = 128 + (nChr >>> 18 & 63);
      aBytes[nIdx++] = 128 + (nChr >>> 12 & 63);
      aBytes[nIdx++] = 128 + (nChr >>> 6 & 63);
      aBytes[nIdx++] = 128 + (nChr & 63);
    }
  }
  return aBytes;
}

// src/rawdata/js/lib/markdown_upload.ts
function setupMarkdownUpload(eSubmits, eFileInput, eUploadBar, eText, doMarkdown, maxFileSize, uploadUrl) {
  const submitTexts = Array.from(eSubmits).map((e) => e.value);
  const uploadProgress = must(eUploadBar.querySelector(".progress"));
  const uploadProgressText = must(eUploadBar.querySelector(".progress_text"));
  const uploadProgressBar = must(eUploadBar.querySelector(".progress_bar"));
  const uploadProgressBarFill = must(eUploadBar.querySelector(".progress_bar > div"));
  let fileCounter = 0;
  let enterCounter = 0;
  let uploadQueue = [];
  let currentUpload = null;
  let currentXhr = null;
  let currentBatchSize = 0;
  let currentBatchDone = 0;
  eFileInput.addEventListener("change", () => {
    if (eFileInput.files.length > 0) {
      importUserFiles(eFileInput.files);
    }
  });
  eText.addEventListener("dragover", (ev) => {
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
  eText.addEventListener("dragenter", function(ev) {
    enterCounter++;
    let droppable = false;
    for (let i = 0; i < ev.dataTransfer.items.length; ++i) {
      if (ev.dataTransfer.items[i].kind.toLowerCase() == "file") {
        droppable = true;
        break;
      }
    }
    if (droppable) {
      eText.classList.add("drop");
    }
  });
  eText.addEventListener("dragleave", function(ev) {
    enterCounter--;
    if (enterCounter == 0) {
      eText.classList.remove("drop");
    }
  });
  function makeUploadString(uploadNumber, filename) {
    return `Uploading file #${uploadNumber}: \`${filename}\`...`;
  }
  eText.addEventListener("drop", (ev) => {
    enterCounter = 0;
    eText.classList.remove("drop");
    if (ev.dataTransfer && ev.dataTransfer.files) {
      importUserFiles(ev.dataTransfer.files);
    }
    ev.preventDefault();
  });
  eText.addEventListener("paste", (ev) => {
    const files = ev.clipboardData?.files;
    if (files?.length ?? 0 > 0) {
      importUserFiles(files);
      ev.preventDefault();
    }
  });
  function importUserFiles(files) {
    let items = [];
    for (let i = 0; i < files.length; ++i) {
      let f = files[i];
      if (f.size < maxFileSize) {
        items.push({ file: f, error: null });
      } else {
        items.push({ file: null, error: `\`${f.name}\` is too big! Max size is ${maxFileSize} but the file is ${f.size}.` });
      }
    }
    let cursorStart = eText.selectionStart;
    let cursorEnd = eText.selectionEnd;
    let toInsert = "";
    let linesToCursor = eText.value.substr(0, cursorStart).split("\n");
    let cursorLine = linesToCursor[linesToCursor.length - 1].trim();
    if (cursorLine.length > 0) {
      toInsert = "\n\n";
    }
    for (const item of items) {
      if (item.file) {
        fileCounter++;
        toInsert += makeUploadString(fileCounter, item.file.name) + "\n\n";
        queueUpload(fileCounter, item.file);
      } else {
        toInsert += `${item.error}

`;
      }
    }
    eText.value = eText.value.substring(0, cursorStart) + toInsert + eText.value.substring(cursorEnd, eText.value.length);
    eText.selectionStart = cursorStart + toInsert.length;
    eText.selectionEnd = eText.selectionStart;
    doMarkdown();
    uploadNext();
  }
  function replaceUploadString(upload, newString) {
    let cursorStart = eText.selectionStart;
    let cursorEnd = eText.selectionEnd;
    let uploadString = makeUploadString(upload.uploadNumber, upload.file.name);
    let insertIndex = eText.value.indexOf(uploadString);
    if (insertIndex === -1) {
      insertIndex = eText.value.length;
      const newLines = newString.startsWith("\n\n") ? "" : "\n\n";
      eText.value = eText.value + newLines + newString;
    } else {
      eText.value = eText.value.replace(uploadString, newString);
    }
    const intersects = cursorStart < insertIndex + uploadString.length && insertIndex < cursorEnd;
    const fullyInside = insertIndex <= cursorStart && cursorEnd <= insertIndex + uploadString.length;
    if (fullyInside && cursorStart === cursorEnd || intersects && !fullyInside) {
      eText.selectionStart = eText.selectionEnd = insertIndex + newString.length;
    } else {
      const difference = newString.length - uploadString.length;
      eText.selectionStart = cursorStart >= insertIndex + uploadString.length ? cursorStart + difference : cursorStart;
      eText.selectionEnd = cursorEnd >= insertIndex + uploadString.length ? cursorEnd + difference : cursorEnd;
    }
    doMarkdown();
  }
  function replaceUploadStringError(upload) {
    replaceUploadString(upload, `There was a problem uploading your file \`${upload.file.name}\`.`);
  }
  function queueUpload(uploadNumber, file) {
    uploadQueue.push({
      uploadNumber,
      file
    });
    currentBatchSize++;
    uploadProgressText.textContent = `Uploading files ${currentBatchDone + 1}/${currentBatchSize}`;
  }
  function uploadDone(ev) {
    assert(currentXhr);
    assert(currentUpload);
    try {
      if (currentXhr.status == 200 && currentXhr.response) {
        if (currentXhr.response.url) {
          let url = currentXhr.response.url;
          let newString = `[${currentUpload.file.name}](${url})`;
          if (currentXhr.response.mime.startsWith("image")) {
            newString = "!" + newString;
          }
          replaceUploadString(currentUpload, newString);
        } else if (currentXhr.response.error) {
          replaceUploadString(currentUpload, `Upload failed for \`${currentUpload.file.name}\`: ${currentXhr.response.error}.`);
        } else {
          replaceUploadStringError(currentUpload);
        }
      } else {
        replaceUploadStringError(currentUpload);
      }
    } catch (err) {
      console.error(err);
      replaceUploadStringError(currentUpload);
    }
    currentUpload = null;
    currentXhr = null;
    currentBatchDone++;
    uploadNext();
  }
  function updateUploadProgress(ev) {
    if (ev.lengthComputable) {
      let progress = ev.loaded / ev.total;
      uploadProgressBarFill.style.width = Math.floor(progress * 100) + "%";
    }
  }
  function uploadNext() {
    if (currentUpload == null) {
      const next = uploadQueue.shift();
      if (next) {
        uploadProgressText.textContent = `Uploading files ${currentBatchDone + 1}/${currentBatchSize}`;
        eUploadBar.classList.add("uploading");
        uploadProgressBarFill.style.width = "0%";
        for (const e of eSubmits) {
          e.disabled = true;
          e.value = "Uploading files...";
        }
        try {
          let utf8Filename = strToUTF8Arr(next.file.name);
          let base64Filename = base64EncArr(utf8Filename);
          currentXhr = new XMLHttpRequest();
          currentXhr.upload.addEventListener("progress", updateUploadProgress);
          currentXhr.open("POST", uploadUrl, true);
          currentXhr.setRequestHeader("Hmn-Upload-Filename", base64Filename);
          currentXhr.responseType = "json";
          currentXhr.addEventListener("loadend", uploadDone);
          currentXhr.send(next.file);
          currentUpload = next;
        } catch (err) {
          replaceUploadStringError(next);
          console.error(err);
          uploadNext();
        }
      } else {
        for (const [i, e] of Array.from(eSubmits).entries()) {
          e.disabled = false;
          e.value = submitTexts[i];
        }
        eUploadBar.classList.remove("uploading");
        currentBatchSize = 0;
        currentBatchDone = 0;
      }
    }
  }
}

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

// src/rawdata/js/lib/markdown_previews.ts
var previewWorker = new Worker("/assets/markdown_worker.js");
function autosaveContent({
  inputEl,
  storageKey
}) {
  const storagePrefix = "saved-content";
  const aWeekAgo = (/* @__PURE__ */ new Date()).getTime() - 7 * 24 * 60 * 60 * 1e3;
  for (const key in window.localStorage) {
    if (!window.localStorage.hasOwnProperty(key)) {
      continue;
    }
    if (key.startsWith(storagePrefix)) {
      try {
        const { when } = JSON.parse(window.localStorage.getItem(key));
        if (when <= aWeekAgo) {
          window.localStorage.removeItem(key);
        }
      } catch (e) {
        console.error(e);
      }
    }
  }
  const storageKeyFull = `${storagePrefix}/${storageKey}`;
  const storedContents = window.localStorage.getItem(storageKeyFull);
  if (storedContents && !inputEl.value) {
    try {
      const { contents } = JSON.parse(storedContents);
      inputEl.value = contents;
    } catch (e) {
      console.error(e);
    }
  }
  function updateContentCache() {
    window.localStorage.setItem(storageKeyFull, JSON.stringify({
      when: (/* @__PURE__ */ new Date()).getTime(),
      contents: inputEl.value
    }));
  }
  inputEl.addEventListener("input", () => updateContentCache());
  return {
    clear() {
      window.localStorage.removeItem(storageKeyFull);
    }
  };
}
var markdownIds = [];
function initLiveMarkdown({
  inputEl,
  previewEl,
  parserName
}) {
  if (!parserName) {
    parserName = "parseMarkdown";
  }
  if (markdownIds.includes(inputEl.id)) {
    console.warn(`Multiple elements with ID "${inputEl.id}" are being used for Markdown. Results will be very confusing!`);
  }
  markdownIds.push(inputEl.id);
  previewWorker.onmessage = ({ data }) => {
    const { elementID, html } = data;
    if (elementID === inputEl.id) {
      previewEl.innerHTML = html;
      MathJax.typeset?.();
    }
  };
  function doMarkdown() {
    previewWorker.postMessage({
      elementID: inputEl.id,
      markdown: inputEl.value,
      parserName
    });
  }
  doMarkdown();
  inputEl.addEventListener("input", () => doMarkdown());
  return doMarkdown;
}

// src/rawdata/js/project_edit.ts
function init({
  csrf,
  projectName,
  maxOwners,
  logoMaxFileSize,
  headerMaxFileSize,
  textMaxFileSize,
  editorUploadUrl,
  initialLinks,
  ownerCheckUrl,
  logo,
  headerImage,
  screenshots
}) {
  initHashTabs(document);
  const projectForm = must(document.querySelector("#project_form"));
  const tag = must(document.querySelector("#tag"));
  const tagPreview = must(document.querySelector("#tag-preview"));
  function updateTagPreview() {
    tagPreview.innerText = tag.value || "[your tag]";
  }
  updateTagPreview();
  tag.addEventListener("input", () => updateTagPreview());
  const description = must(document.querySelector("#full_description"));
  const descPreview = must(document.querySelector("#desc_preview"));
  const { clear: clearDescription } = autosaveContent({
    inputEl: description,
    storageKey: `project-description/${projectName}`
  });
  projectForm.addEventListener("submit", () => clearDescription());
  const doMarkdown = initLiveMarkdown({ inputEl: description, previewEl: descPreview });
  const OWNER_QUERY_STATE_IDLE = 0;
  const OWNER_QUERY_STATE_QUERYING = 1;
  let ownerQueryState = OWNER_QUERY_STATE_IDLE;
  const addOwnerInput = must(document.querySelector("#owner_name"));
  const addOwnerButton = must(document.querySelector("#owner_add"));
  const ownersError = must(document.querySelector("#owners_error"));
  const ownerList = must(document.querySelector("#owner_list"));
  const ownerTemplate = makeTemplateCloner("owner_row");
  const ownerPreviewTemplate = makeTemplateCloner("owner_preview");
  const ownersPreviewContainer = must(document.querySelector("#owners_preview"));
  addOwnerInput.addEventListener("keypress", function(ev) {
    if (ev.which == 13) {
      startAddOwner();
      ev.preventDefault();
      ev.stopPropagation();
    }
  });
  addOwnerButton.addEventListener("click", function(ev) {
    ev.preventDefault();
    startAddOwner();
  });
  function updateAddOwnerStyles() {
    const numOwnerRows = document.querySelectorAll(".owner_row").length;
    addOwnerInput.disabled = numOwnerRows >= maxOwners;
  }
  updateAddOwnerStyles();
  function startAddOwner() {
    if (ownerQueryState == OWNER_QUERY_STATE_QUERYING) {
      return;
    }
    let newOwner = addOwnerInput.value.trim().toLowerCase();
    if (newOwner.length == 0) {
      return;
    }
    let ownerEls = ownerList.querySelectorAll(".owner_row input[name='owners']");
    for (let i = 0; i < ownerEls.length; ++i) {
      let existingOwner = ownerEls[i].value.toLowerCase();
      if (newOwner == existingOwner) {
        return;
      }
    }
    ownersError.textContent = "";
    let xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open("POST", ownerCheckUrl);
    xhr.responseType = "json";
    xhr.addEventListener("load", function(ev) {
      let result = xhr.response;
      if (result) {
        if (result.found) {
          addOwner(result.username, result.name, result.avatarUrl);
          addOwnerInput.value = "";
        } else {
          ownersError.textContent = "Username not found";
        }
      } else {
        ownersError.textContent = "There was an issue validating this username";
      }
      setOwnerQueryState(OWNER_QUERY_STATE_IDLE);
      if (document.activeElement == addOwnerButton) {
        addOwnerInput.focus();
      }
    });
    xhr.addEventListener("error", function(ev) {
      ownersError.textContent = "There was an issue validating this username";
      setOwnerQueryState(OWNER_QUERY_STATE_IDLE);
    });
    let data = new FormData();
    data.append(csrf.field, csrf.token);
    data.append("username", newOwner);
    xhr.send(data);
    setOwnerQueryState(OWNER_QUERY_STATE_QUERYING);
  }
  function setOwnerQueryState(state) {
    ownerQueryState = state;
    const querying = ownerQueryState == OWNER_QUERY_STATE_QUERYING;
    addOwnerInput.disabled = querying;
    addOwnerButton.disabled = querying;
    updateAddOwnerStyles();
  }
  function addOwner(username, bestName, avatarUrl) {
    let ownerEl = ownerTemplate();
    ownerEl.input.value = username;
    ownerEl.name.textContent = bestName;
    ownerEl.avatar.src = avatarUrl;
    ownerList.appendChild(ownerEl.rootElement);
    updateAddOwnerStyles();
    updateOwnersPreview();
  }
  ownerList.addEventListener("click", (ev) => {
    if (ev.target.closest(".remove_owner")) {
      must(ev.target.closest(".owner_row")).remove();
    }
    updateAddOwnerStyles();
    updateOwnersPreview();
  });
  function updateOwnersPreview() {
    let ownerEls = ownerList.querySelectorAll(".owner_row");
    ownersPreviewContainer.innerHTML = "";
    for (let i = 0; i < ownerEls.length; ++i) {
      let avatarUrl = must(ownerEls[i].querySelector("img")).src;
      let name = must(ownerEls[i].querySelector("span")).textContent;
      let previewEl = ownerPreviewTemplate();
      previewEl.avatar.src = avatarUrl;
      previewEl.name.textContent = name;
      ownersPreviewContainer.appendChild(previewEl.root);
    }
  }
  updateOwnersPreview();
  const projectNameField = must(document.querySelector("#project_name"));
  const descriptionField = must(document.querySelector("#description"));
  const logoSelector = new ImageSelector(
    "logo",
    logoMaxFileSize,
    {
      original: logo || void 0,
      onUpdate() {
        updateCardPreview();
      }
    }
  );
  must(document.querySelector("#logo-upload-button")).addEventListener("click", (e) => {
    e.preventDefault();
    logoSelector.openImageInput();
  });
  must(document.querySelector("#project-logo-placeholder")).replaceWith(logoSelector.root);
  const headerSelector = new ImageSelector(
    "header_image",
    headerMaxFileSize,
    {
      original: headerImage || void 0,
      onUpdate() {
        updateCardPreview();
      }
    }
  );
  must(document.querySelector("#header-upload-button")).addEventListener("click", (e) => {
    e.preventDefault();
    headerSelector.openImageInput();
  });
  must(document.querySelector("#header-image-placeholder")).replaceWith(headerSelector.root);
  function updateCardPreview() {
    const title = projectNameField.value || "Project Title";
    must(document.querySelector("#logo_preview img")).src = logoSelector.url;
    must(document.querySelector("#logo_placeholder")).innerText = title[0].toUpperCase();
    must(document.querySelector("#logo_placeholder")).hidden = !!logoSelector.url;
    must(document.querySelector("#header_img_preview")).style.backgroundImage = `url(${headerSelector.url})`;
    must(document.querySelector("#flowsnake")).classList.toggle("dn", !!headerSelector.url);
    must(document.querySelector("#name_preview")).innerText = title;
    must(document.querySelector("#longdesc_title")).innerText = title;
    must(document.querySelector("#blurb_preview")).innerText = descriptionField.value || "Project summary";
    must(document.querySelector("#logo-selector-container")).hidden = !logoSelector.url;
    must(document.querySelector("#header-image-selector-container")).hidden = !headerSelector.url;
  }
  updateCardPreview();
  projectNameField.addEventListener("input", updateCardPreview);
  descriptionField.addEventListener("input", updateCardPreview);
  setupMarkdownUpload(
    document.querySelectorAll("#project_form input[type=submit]"),
    must(document.querySelector("#file_input")),
    must(document.querySelector(".upload_bar")),
    description,
    doMarkdown,
    textMaxFileSize,
    editorUploadUrl
  );
  initLinkEditor(initialLinks);
  const primaryLinkTemplate = makeTemplateCloner("primary_link");
  const secondaryLinkTemplate = makeTemplateCloner("secondary_link");
  function updateLinkPreviews() {
    const linkData = getLinkData();
    const primaryPreview = must(document.querySelector("#primary_links_preview"));
    const secondaryPreview = must(document.querySelector("#secondary_links_preview"));
    primaryPreview.innerHTML = "";
    secondaryPreview.innerHTML = "";
    for (const link of linkData) {
      if (link.primary) {
        const l = primaryLinkTemplate();
        l.link.href = link.url;
        l.name.innerText = link.name;
        primaryPreview.appendChild(l.link);
      } else {
        let icon = "website";
        let title = "";
        if (parseKnownServicesForUrl) {
          const guess = parseKnownServicesForUrl(link.url);
          icon = guess.icon;
          title = guess.service;
          if (guess.username) {
            title += ` (${guess.username})`;
          }
        }
        const iconSVG = must(document.querySelector(`#link-icon-${icon}`)).innerHTML;
        const l = secondaryLinkTemplate();
        l.link.href = link.url;
        l.link.title = link.name || title;
        l.link.innerHTML = iconSVG;
        secondaryPreview.appendChild(l.link);
      }
    }
  }
  updateLinkPreviews();
  window.addEventListener("wasmready", () => updateLinkPreviews());
  window.addEventListener("linkedit", () => updateLinkPreviews());
  const screenshotContainer = must(document.querySelector("#screenshots"));
  const screenshotTemplate = makeTemplateCloner("screenshot");
  const { startDrag: startDragScreenshot } = initReorderable(screenshotContainer, {
    onReorder() {
    }
  });
  for (const screenshot of screenshots ?? []) {
    const el = screenshotTemplate();
    const selector = new ImageSelector("screenshot", headerMaxFileSize, {
      original: screenshot,
      onRemove: () => {
        el.root.hidden = true;
      }
    });
    el.selectorPlaceholder.replaceWith(selector.root);
    el.grabHandle.addEventListener("pointerdown", startDragScreenshot);
    screenshotContainer.appendChild(el.root);
  }
  const newScreenshotButton = must(document.querySelector("#screenshot-upload-button"));
  newScreenshotButton.addEventListener("click", async (e) => {
    e.preventDefault();
    const el = screenshotTemplate();
    const selector = new ImageSelector("screenshot", headerMaxFileSize, {
      onRemove: () => el.root.remove()
    });
    el.selectorPlaceholder.replaceWith(selector.root);
    el.grabHandle.addEventListener("pointerdown", startDragScreenshot);
    el.root.hidden = true;
    document.body.appendChild(el.root);
    const file = await selector.openImageInput();
    if (file) {
      el.root.remove();
      screenshotContainer.appendChild(el.root);
      el.root.hidden = false;
    } else {
      el.root.remove();
    }
  });
}
export {
  init
};
