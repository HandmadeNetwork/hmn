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
  parserName = "parseMarkdown"
}) {
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

// src/rawdata/js/editor.ts
var form = must(document.querySelector("#form"));
var titleField = document.querySelector("#title");
var textField = must(document.querySelector("#editor"));
var preview = must(document.querySelector("#preview"));
function init({ maxFileSize, uploadUrl }) {
  const clearFuncs = [];
  if (titleField) {
    const { clear: clearTitle } = autosaveContent({
      inputEl: titleField,
      storageKey: `post-title/${window.location.host}${window.location.pathname}`
    });
    clearFuncs.push(clearTitle);
  }
  const { clear: clearContent } = autosaveContent({
    inputEl: textField,
    storageKey: `post-content/${window.location.host}${window.location.pathname}`
  });
  clearFuncs.push(clearContent);
  form.addEventListener("submit", (e) => {
    for (const clear of clearFuncs) {
      clear();
    }
  });
  const doMarkdown = initLiveMarkdown({ inputEl: textField, previewEl: preview });
  setupMarkdownUpload(
    document.querySelectorAll("#form input[type=submit]"),
    must(document.querySelector("#file_input")),
    must(document.querySelector(".upload_bar")),
    textField,
    doMarkdown,
    maxFileSize,
    uploadUrl
  );
  textField.addEventListener("keydown", function(ev) {
    if (ev.ctrlKey) {
      const key = ev.key.toLowerCase();
      const start = textField.selectionStart;
      const end = textField.selectionEnd;
      const selected = textField.value.substring(start, end);
      if (key === "k") {
        ev.preventDefault();
        if (selected.length) {
          textField.value = textField.value.substring(0, start) + "[" + selected + "]()" + textField.value.substring(end);
          textField.selectionStart = textField.selectionEnd = end + 3;
        } else {
          textField.value = textField.value.substring(0, start) + "[](url)" + textField.value.substring(end);
          textField.selectionStart = textField.selectionEnd = start + 1;
        }
        textField.focus();
        doMarkdown();
      } else if (key === "b") {
        ev.preventDefault();
        if (selected.length) {
          textField.value = textField.value.substring(0, start) + "**" + selected + "**" + textField.value.substring(end);
          textField.selectionStart = textField.selectionEnd = end + 2;
        } else {
          textField.value = textField.value.substring(0, start) + "****" + textField.value.substring(end);
          textField.selectionStart = textField.selectionEnd = start + 2;
        }
        textField.focus();
        doMarkdown();
      } else if (key === "i") {
        ev.preventDefault();
        if (selected.length) {
          textField.value = textField.value.substring(0, start) + "*" + selected + "*" + textField.value.substring(end);
          textField.selectionStart = textField.selectionEnd = end + 1;
        } else {
          textField.value = textField.value.substring(0, start) + "**" + textField.value.substring(end);
          textField.selectionStart = textField.selectionEnd = start + 1;
        }
        textField.focus();
        doMarkdown();
      } else if (key === "s") {
        ev.preventDefault();
      }
    }
  });
  textField.addEventListener("paste", (ev) => {
    const clipboard = must(ev.clipboardData);
    const pastedText = clipboard.getData("text");
    const start = textField.selectionStart;
    const end = textField.selectionEnd;
    if (pastedText && (pastedText.startsWith("https://") || pastedText.startsWith("http://")) && start !== end) {
      ev.preventDefault();
      textField.value = textField.value.substring(0, start) + "[" + textField.value.substring(start, end) + "](" + pastedText + ")" + textField.value.substring(end);
      textField.selectionStart = textField.selectionEnd = end + 4 + pastedText.length;
      textField.focus();
      doMarkdown();
    }
  });
}
export {
  init
};
