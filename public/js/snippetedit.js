const snippetEditTemplate = makeTemplateCloner("snippet-edit");
const snippetEditProjectTemplate = makeTemplateCloner("snippet-edit-project");

function readableByteSize(numBytes) {
	const scales = [
		" bytes",
		"kb",
		"mb",
		"gb"
	];
	let scale = 0;
	while (numBytes > 1024 && scale < scales.length-1) {
		numBytes /= 1024;
		scale++;
	}
	return new Intl.NumberFormat([], { maximumFractionDigits: (scale > 0 ? 2 : 0) }).format(numBytes) + scales[scale];
}

function makeSnippetEdit(ownerName, ownerAvatar, ownerUrl, date, text, attachmentElement, projectIds, stickyProjectId, snippetId, originalSnippetEl) {
	let snippetEdit = snippetEditTemplate();
	let projectSelector = null;
	let originalAttachment = null;
	let originalText = text;
	let attachmentChanged = false;
	let hasAttachment = false;
	snippetEdit.redirect.value = location.href;
	snippetEdit.avatarImg.src = ownerAvatar;
	snippetEdit.avatarLink.href = ownerUrl;
	snippetEdit.username.textContent = ownerName;
	snippetEdit.username.href = ownerUrl;
	snippetEdit.date.textContent = new Intl.DateTimeFormat([], {month: "2-digit", day: "2-digit", year: "numeric"}).format(date);
	snippetEdit.text.value = text;
	if (attachmentElement) {
		originalAttachment = attachmentElement.cloneNode(true);
		clearAttachment(true);
	}
	if (snippetId !== undefined && snippetId !== null) {
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
		projEl.projectId.value = proj.id;
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
			projectSelector = document.createElement("SELECT");
			let option = document.createElement("OPTION");
			option.textContent = "Add to project...";
			option.selected = true;
			projectSelector.appendChild(option);
			for (let i = 0; i < remainingProjects.length; ++i) {
				option = document.createElement("OPTION");
				option.value = remainingProjects[i].id;
				option.selected = false;
				option.textContent = remainingProjects[i].name;
				projectSelector.appendChild(option);
			}
			projectSelector.addEventListener("change", function(ev) {
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
			snippetEdit.uploadResetLink.style.display = "none";
			snippetEdit.previewContent = emptyElement(snippetEdit.previewContent);
			snippetEdit.previewContent.appendChild(el);
			snippetEdit.resetLink.style.display = (!originalAttachment || el == originalAttachment) ? "none" : "inline-block";
		} else {
			snippetEdit.uploadBox.style.display = "flex";
			snippetEdit.previewBox.style.display = "none";
			if (originalAttachment) {
				snippetEdit.uploadResetLink.style.display = "block";
			}
		}
	}

	function validate() {
		let sizeGood = true;
		if (snippetEdit.file.files.length > 0 && snippetEdit.file.files[0].size > maxFilesize) {
			// NOTE(asaf): Writing this out in bytes to make the limit exactly clear to the user.
			let readableSize = new Intl.NumberFormat([], { useGrouping: "always" }).format(maxFilesize);
			snippetEdit.errors.textContent = "File is too big! Max filesize is " + readableSize + " bytes";
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

	snippetEdit.uploadLink.addEventListener("click", function() {
		snippetEdit.file.click();
	});

	snippetEdit.removeLink.addEventListener("click", function() {
		clearAttachment(false);
	});

	snippetEdit.replaceLink.addEventListener("click", function() {
		snippetEdit.file.click();
	});

	snippetEdit.resetLink.addEventListener("click", function() {
		clearAttachment(true);
	});

	snippetEdit.uploadResetLink.addEventListener("click", function() {
		clearAttachment(true);
	});

	snippetEdit.file.addEventListener("change", function() {
		if (snippetEdit.file.files.length > 0) {
			setFile(snippetEdit.file.files[0]);
		}
	});

	snippetEdit.root.addEventListener("dragover", function(ev) {
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

	snippetEdit.root.addEventListener("dragenter", function(ev) {
		enterCounter++;
		if (ev.dataTransfer && ev.dataTransfer.files && ev.dataTransfer.files.length > 0) {
			snippetEdit.root.classList.add("drop");
		}
	});

	snippetEdit.root.addEventListener("dragleave", function(ev) {
		enterCounter--;
		if (enterCounter == 0) {
			snippetEdit.root.classList.remove("drop");
		}
	});

	snippetEdit.root.addEventListener("drop", function(ev) {
		enterCounter = 0;
		snippetEdit.root.classList.remove("drop");

		if (ev.dataTransfer && ev.dataTransfer.files && ev.dataTransfer.files.length > 0) {
			setFile(ev.dataTransfer.files[0]);
		}

		ev.preventDefault();
	});

	snippetEdit.text.addEventListener("paste", function(ev) {
		const files = ev.clipboardData?.files ?? [];
		if (files.length > 0) {
			setFile(files[0]);
		}
	});

	snippetEdit.text.addEventListener("input", function(ev) {
		validate();
	});

	snippetEdit.saveButton.addEventListener("click", function(ev) {
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
			// NOTE(asaf): We're in edit mode and nothing changed, so no need to submit to the server.
			ev.preventDefault();
			cancel();
		}
	});
	
	snippetEdit.deleteButton.addEventListener("click", function(ev) {
		snippetEdit.file.value = "";
	});

	validate();

	return snippetEdit;
}

function editTimelineSnippet(timelineItemEl, stickyProjectId) {
	let ownerName = timelineItemEl.querySelector(".user")?.textContent;
	let ownerUrl = timelineItemEl.querySelector(".user")?.href;
	let ownerAvatar = timelineItemEl.querySelector(".avatar-icon")?.src;
	let creationDate = new Date(timelineItemEl.querySelector("time").dateTime);
	let rawDesc = timelineItemEl.querySelector(".rawdesc").textContent;
	let attachment = timelineItemEl.querySelector(".timeline-content-box")?.children?.[0];
	let projectIds = [];
	let projectEls = timelineItemEl.querySelectorAll(".projects > a");
	for (let i = 0; i < projectEls.length; ++i) {
		let projid = projectEls[i].getAttribute("data-projid");
		if (projid) {
			projectIds.push(projid);
		}
	}
	let snippetEdit = makeSnippetEdit(ownerName, ownerAvatar, ownerUrl, creationDate, rawDesc, attachment, projectIds, stickyProjectId, timelineItemEl.getAttribute("data-id"), timelineItemEl);
	timelineItemEl.parentElement.insertBefore(snippetEdit.root, timelineItemEl);
	timelineItemEl.remove();
}
