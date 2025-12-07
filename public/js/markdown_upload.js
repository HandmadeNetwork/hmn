// Requires base64.js

/**
 * Sets up file / image uploading for Markdown content.
 *
 * @param eSubmits A list of elements of buttons to submit/save the page you're on. They
 *                 will be disabled and tell users files are uploading while uploading is
 *                 happening.
 * @param eFileInput The `<input type="file">`
 * @param eUploadBar Usually looks like
 *                   ```
 *                   <div class="upload_bar flex-grow-1">
 *                     <div class="instructions">
 *                       Upload files by dragging & dropping, pasting, or <label class="pointer link" for="file_input">selecting</label> them.
 *                     </div>
 *                     <div class="progress flex">
 *                       <div class="progress_text mr3"></div>
 *                       <div class="progress_bar flex-grow-1 flex-shrink-1 pa1"><div class=""></div></div>
 *                     </div>
 *                   </div>
 *                   ```
 * @param eText The text field that can be dropped into and is editing the markdown.
 * @param doMarkdown The function returned by `initLiveMarkdown`.
 * @param maxFileSize The max allowed file size in bytes.
 * @param uploadUrl The URL to POST assets to (unique per project to avoid CORS issues).
 */
function setupMarkdownUpload(eSubmits, eFileInput, eUploadBar, eText, doMarkdown, maxFileSize, uploadUrl) {
	const submitTexts = Array.from(eSubmits).map(e => e.value);
	const uploadProgress = eUploadBar.querySelector('.progress');
	const uploadProgressText = eUploadBar.querySelector('.progress_text');
	const uploadProgressBar = eUploadBar.querySelector('.progress_bar');
	const uploadProgressBarFill = eUploadBar.querySelector('.progress_bar > div');
	let fileCounter = 0;
	let enterCounter = 0;
	let uploadQueue = [];
	let currentUpload = null;
	let currentXhr = null;
	let currentBatchSize = 0;
	let currentBatchDone = 0;

	eFileInput.addEventListener("change", function(ev) {
		if (eFileInput.files.length > 0) {
			importUserFiles(eFileInput.files);
		}
	});

	eText.addEventListener("dragover", function(ev) {
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

	eText.addEventListener("drop", function(ev) {
		enterCounter = 0;
		eText.classList.remove("drop");

		if (ev.dataTransfer && ev.dataTransfer.files) {
			importUserFiles(ev.dataTransfer.files)
		}

		ev.preventDefault();
	});

	eText.addEventListener("paste", function(ev) {
		const files = ev.clipboardData?.files ?? [];
		if (files.length > 0) {
			importUserFiles(files)
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
		let cursorLine = linesToCursor[linesToCursor.length-1].trim();
		if (cursorLine.length > 0) {
			toInsert = "\n\n";
		}
		for (let i = 0; i < items.length; ++i) {
			if (items[i].file) {
				fileCounter++;
				toInsert += makeUploadString(fileCounter, items[i].file.name) + "\n\n";
				queueUpload(fileCounter, items[i].file);
			} else {
				toInsert += `${items[i].error}\n\n`;
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
		let insertIndex = eText.value.indexOf(uploadString)
		
		// The user deleted part of the upload string during the upload.
		// Paste the newString at the end.
		if (insertIndex === -1) {
			insertIndex = eText.value.length;
			const newLines = newString.startsWith('\n\n') ? '' : '\n\n';
			eText.value = eText.value + newLines + newString;
		}
		else {
			eText.value = eText.value.replace(uploadString, newString);
		}

		const intersects = cursorStart < insertIndex + uploadString.length && insertIndex < cursorEnd;
		const fullyInside = insertIndex <= cursorStart && cursorEnd <= insertIndex + uploadString.length;
		if ( (fullyInside && cursorStart === cursorEnd) || (intersects && !fullyInside) ) {
			// The user's cursor is inside the placeholder string, or some but not all of the placeholder is selected
			// the cursor should be moved to the end of the replaced string
			eText.selectionStart = eText.selectionEnd = insertIndex + newString.length;
		}
		else {
			// Common case: The user's cursor / selection is outside the placeholder.
			const difference = newString.length - uploadString.length;
			eText.selectionStart = cursorStart >= insertIndex + uploadString.length
				? cursorStart + difference
				: cursorStart;

			eText.selectionEnd = cursorEnd >= insertIndex + uploadString.length
				? cursorEnd + difference
				: cursorEnd;
		}

		doMarkdown();
	}

	function replaceUploadStringError(upload) {
		replaceUploadString(upload, `There was a problem uploading your file \`${upload.file.name}\`.`);
	}

	function queueUpload(uploadNumber, file) {
		uploadQueue.push({
			uploadNumber: uploadNumber,
			file: file
		});

		currentBatchSize++;
		uploadProgressText.textContent = `Uploading files ${currentBatchDone+1}/${currentBatchSize}`;
	}

	function uploadDone(ev) {
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
			next = uploadQueue.shift();
			if (next) {
				uploadProgressText.textContent = `Uploading files ${currentBatchDone+1}/${currentBatchSize}`;
				eUploadBar.classList.add("uploading");
				uploadProgressBarFill.style.width = "0%";
				for (const e of eSubmits) {
					e.disabled = true;
					e.value = "Uploading files...";
				}

				try {
					let utf8Filename = strToUTF8Arr(next.file.name);
					let base64Filename = base64EncArr(utf8Filename);
					// NOTE(asaf): We use XHR because fetch can't do upload progress reports. Womp womp. https://youtu.be/Pubd-spHN-0?t=2
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
				for (const [i, e] of eSubmits.entries()) {
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
