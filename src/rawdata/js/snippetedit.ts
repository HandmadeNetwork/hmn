import { SnippetEditAvailableProject, SnippetEditorConfig } from "./lib/models";
import { emptyElement, makeTemplateCloner } from "./lib/templates";
import { HTMLFileInputElement } from "./lib/types";
import { assert, must } from "./lib/utils";

const snippetEditorTemplate = makeTemplateCloner<{
	root: HTMLFormElement,
	redirect: HTMLInputElement,
	snippetId: HTMLInputElement,
	removeAttachment: HTMLInputElement,
	file: HTMLFileInputElement,
	avatarLink: HTMLAnchorElement,
	avatarImg: HTMLImageElement,
	username: HTMLAnchorElement,
	date: HTMLElement,
	cancelLink: HTMLAnchorElement,
	text: HTMLTextAreaElement,
	uploadBox: HTMLElement,
	uploadLink: HTMLAnchorElement,
	uploadResetBox: HTMLElement,
	uploadResetLink: HTMLAnchorElement,
	previewBox: HTMLElement,
	previewContent: HTMLElement,
	removeLink: HTMLAnchorElement,
	resetLink: HTMLAnchorElement,
	replaceLink: HTMLAnchorElement,
	errors: HTMLElement,
	projectList: HTMLElement,
	deleteButton: HTMLInputElement,
	saveButton: HTMLInputElement,
}>("snippet-edit");
const snippetEditProjectTemplate = makeTemplateCloner<{
	root: HTMLElement,
	projectId: HTMLInputElement,
	projectLogo: HTMLImageElement,
	projectName: HTMLElement,
	removeButton: HTMLAnchorElement,
}>("snippet-edit-project");

function readableByteSize(numBytes: number) {
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
	return new Intl.NumberFormat([], { maximumFractionDigits: (scale > 0 ? 2 : 0) }).format(numBytes) + scales[scale];
}

export type SnippetEditOptions = {
	config: SnippetEditorConfig,
	edit?: SnippetEditorEditConfig,
};

export type SnippetEditorEditConfig = {
	el: HTMLElement,
	id: string,
	creationDate: Date,
	text: string,
	projectIDs: number[],
}

// Instantiates a snippet editor and returns the template element. The `config`
// is expected to be passed directly from Go code. If you have an existing
// snippet to edit, that data typically comes from JS, so that is a separate
// param.
export function makeSnippetEdit({
	config,
	edit,
}: SnippetEditOptions) {
	const snippetEditor = snippetEditorTemplate();

	// NOTE(ben): Will be updated during editing by updateProjectSelector, and
	// will be hidden if empty.
	let projectSelector: HTMLSelectElement = document.createElement("select");

	// NOTE(ben): The original media displayed on the snippet, which we need to
	// be able to restore during editing.
	const originalAttachment: Element | undefined = edit?.el.querySelector<HTMLElement>(".timeline-media")?.children?.[0].cloneNode(true) as Element;
	const originalText = edit?.text ?? "";
	let attachmentChanged = false;
	let hasAttachment = false;

	snippetEditor.root.action = config.submitUrl;
	snippetEditor.redirect.value = location.href;
	if (config.owner.avatarUrl) {
		snippetEditor.avatarImg.src = config.owner.avatarUrl;
		snippetEditor.avatarLink.href = config.owner.profileUrl;
		snippetEditor.avatarImg.hidden = false;
	} else {
		snippetEditor.avatarImg.hidden = true;
	}
	snippetEditor.username.textContent = config.owner.name;
	snippetEditor.username.href = config.owner.profileUrl;
	snippetEditor.date.textContent = new Intl.DateTimeFormat([], { month: "2-digit", day: "2-digit", year: "numeric" })
		.format(edit?.creationDate ?? new Date());
	snippetEditor.text.value = originalText;
	if (originalAttachment) {
		clearAttachment(true);
	}
	if (edit) {
		snippetEditor.snippetId.value = edit.id;
	} else {
		snippetEditor.deleteButton.remove();
	}

	if (config.requiredProjectID) {
		const proj = must(
			config.availableProjects.find(p => p.id === config.requiredProjectID),
			"the required project should always be in the list of available projects",
		);
		addProject(proj);
	}
	for (const projectID of edit?.projectIDs ?? []) {
		if (projectID === config.requiredProjectID) {
			continue;
		}
		const proj = config.availableProjects.find(p => p.id === projectID);
		if (proj) {
			addProject(proj);
		}
	}
	updateProjectSelector();

	if (edit?.el) {
		snippetEditor.cancelLink.addEventListener("click", function () {
			cancel();
		});
	} else {
		snippetEditor.cancelLink.remove();
	}

	function cancel() {
		if (edit?.el) {
			snippetEditor.root.parentElement!.insertBefore(edit.el, snippetEditor.root);
		}
		snippetEditor.root.remove();
	}

	function addProject(proj: SnippetEditAvailableProject) {
		let projEl = snippetEditProjectTemplate();
		projEl.projectId.value = `${proj.id}`;
		projEl.projectLogo.src = proj.logo;
		projEl.projectLogo.hidden = !proj.logo;
		projEl.projectName.textContent = proj.name;
		if (proj.id === config.requiredProjectID) {
			projEl.removeButton.remove();
		} else {
			projEl.removeButton.addEventListener("click", function (ev) {
				projEl.root.remove();
				updateProjectSelector();
			});
		}
		snippetEditor.projectList.appendChild(projEl.root);
	}

	// NOTE(ben): Replaces the existing project selector dropdown with a new one
	// reflecting the current list of available projects.
	function updateProjectSelector() {
		projectSelector.remove();

		// NOTE(ben): Look at the contents of the DOM to find which projects have
		// already been assigned and build a list of the remainder.
		const remainingProjects = [];
		const projInputs = snippetEditor.projectList.querySelectorAll<HTMLInputElement>("input[name=project_id]");
		const assignedIds = [];
		for (const projInput of projInputs) {
			let id = parseInt(projInput.value, 10);
			if (!isNaN(id)) {
				assignedIds.push(id);
			}
		}
		for (const project of config.availableProjects) {
			if (!assignedIds.find(id => id === project.id)) {
				remainingProjects.push(project);
			}
		}

		// NOTE(ben): Create and configure a new project selector.
		projectSelector = document.createElement("select");
		projectSelector.hidden = remainingProjects.length === 0;
		{
			const defaultOption = document.createElement("option");
			defaultOption.textContent = "Add to project...";
			defaultOption.selected = true;
			projectSelector.appendChild(defaultOption);

			for (let i = 0; i < remainingProjects.length; ++i) {
				const option = document.createElement("option");
				option.textContent = remainingProjects[i].name;
				option.value = `${remainingProjects[i].id}`;
				option.selected = false;
				projectSelector.appendChild(option);
			}

			projectSelector.addEventListener("change", ev => {
				if (projectSelector.selectedOptions.length === 0) {
					return;
				}
				const selected = projectSelector.selectedOptions[0];
				if (selected.value === "") {
					return;
				}
				const selectedID = parseInt(selected.value, 10);
				if (isNaN(selectedID)) {
					return;
				}

				for (const proj of config.availableProjects) {
					if (proj.id == selectedID) {
						addProject(proj);
						break;
					}
				}
				updateProjectSelector();
			});
		}
		snippetEditor.projectList.appendChild(projectSelector);
	}

	function setFile(file: File) {
		let dt = new DataTransfer();
		dt.items.add(file);
		snippetEditor.file.files = dt.files;

		attachmentChanged = true;
		snippetEditor.removeAttachment.value = "false";
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

	// NOTE(ben): Clears/resets the attached media. If `restoreOriginal` is true,
	// the original media will be put back and nothing will change on submit.
	// Otherwise, media will be removed on submit.
	function clearAttachment(restoreOriginal: boolean) {
		snippetEditor.file.value = "";
		snippetEditor.removeAttachment.value = "false";
		attachmentChanged = false;
		hasAttachment = false;
		let el: Element | undefined = undefined;

		if (originalAttachment) {
			if (restoreOriginal) {
				hasAttachment = true;
				el = originalAttachment;
			} else {
				attachmentChanged = true;
				snippetEditor.removeAttachment.value = "true";
			}
		}

		setPreview(el);
		validate();
	}

	function setPreview(el: Element | undefined) {
		if (el) {
			snippetEditor.uploadBox.style.display = "none";
			snippetEditor.previewBox.style.display = "block";
			snippetEditor.uploadResetBox.style.display = "none";
			snippetEditor.previewContent = emptyElement(snippetEditor.previewContent);
			snippetEditor.previewContent.appendChild(el);
			snippetEditor.resetLink.style.display = (!originalAttachment || el == originalAttachment) ? "none" : "inline-block";
		} else {
			snippetEditor.uploadBox.style.display = "block";
			snippetEditor.previewBox.style.display = "none";
			if (originalAttachment) {
				snippetEditor.uploadResetBox.style.display = "block";
			}
		}
	}

	function validate() {
		let sizeGood = true;
		if (snippetEditor.file.files.length > 0 && snippetEditor.file.files[0].size > config.assetMaxSize) {
			// NOTE(asaf): Writing this out in bytes to make the limit exactly clear to the user.
			let readableSize = new Intl.NumberFormat([], { useGrouping: true }).format(config.assetMaxSize);
			snippetEditor.errors.textContent = "File is too big! Max filesize is " + readableSize + " bytes.";
			sizeGood = false;
		} else {
			snippetEditor.errors.textContent = "";
		}

		let hasText = snippetEditor.text.value.trim().length > 0;

		if ((hasText || hasAttachment) && sizeGood) {
			snippetEditor.saveButton.disabled = false;
		} else {
			snippetEditor.saveButton.disabled = true;
		}
	}

	snippetEditor.uploadLink.addEventListener("click", () => {
		snippetEditor.file.click();
	});

	snippetEditor.removeLink.addEventListener("click", () => {
		clearAttachment(false);
	});

	snippetEditor.replaceLink.addEventListener("click", () => {
		snippetEditor.file.click();
	});

	snippetEditor.resetLink.addEventListener("click", () => {
		clearAttachment(true);
	});

	snippetEditor.uploadResetLink.addEventListener("click", () => {
		clearAttachment(true);
	});

	snippetEditor.file.addEventListener("change", () => {
		if (snippetEditor.file.files.length > 0) {
			setFile(snippetEditor.file.files[0]);
		}
	});

	snippetEditor.root.addEventListener("dragover", ev => {
		assert(ev.dataTransfer);
		let effect: DataTransfer["dropEffect"] = "none";
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

	snippetEditor.root.addEventListener("dragenter", ev => {
		assert(ev.dataTransfer);
		enterCounter++;
		const droppable = Array.from(ev.dataTransfer.items).some(
			item => item.kind.toLowerCase() === "file"
		);
		if (droppable) {
			snippetEditor.root.classList.add("drop");
		}
	});

	snippetEditor.root.addEventListener("dragleave", ev => {
		enterCounter--;
		if (enterCounter == 0) {
			snippetEditor.root.classList.remove("drop");
		}
	});

	snippetEditor.root.addEventListener("drop", ev => {
		enterCounter = 0;
		snippetEditor.root.classList.remove("drop");

		if (ev.dataTransfer && ev.dataTransfer.files && ev.dataTransfer.files.length > 0) {
			setFile(ev.dataTransfer.files[0]);
		}

		ev.preventDefault();
	});

	snippetEditor.text.addEventListener("paste", ev => {
		assert(ev.clipboardData);
		const files = ev.clipboardData.files ?? [];
		if (files.length > 0) {
			setFile(files[0]);
		}
	});

	snippetEditor.text.addEventListener("input", () => {
		validate();
	});

	snippetEditor.saveButton.addEventListener("click", ev => {
		let projectsChanged = false;
		let projInputs = snippetEditor.projectList.querySelectorAll<HTMLInputElement>("input[name=project_id]");
		let assignedIds = [];
		for (let i = 0; i < projInputs.length; ++i) {
			let id = parseInt(projInputs[i].value, 10);
			if (!isNaN(id)) {
				assignedIds.push(id);
			}
		}
		if (edit?.projectIDs.length != assignedIds.length) {
			projectsChanged = true;
		} else {
			for (let i = 0; i < edit?.projectIDs.length; ++i) {
				let found = false;
				for (let j = 0; j < assignedIds.length; ++j) {
					if (edit.projectIDs[i] == assignedIds[j]) {
						found = true;
					}
				}
				if (!found) {
					projectsChanged = true;
					break;
				}
			}
		}

		if (edit && (!attachmentChanged && originalText == snippetEditor.text.value.trim() && !projectsChanged)) {
			// NOTE(asaf): We're in edit mode and nothing changed, so no need to submit to the server.
			ev.preventDefault();
			cancel();
		}
	});

	snippetEditor.deleteButton.addEventListener("click", function (ev) {
		if (!window.confirm("Are you sure you want to delete this snippet?")) {
			ev.preventDefault();
			return;
		}

		snippetEditor.redirect.value = config.onDeleteRedirectUrl ?? "";
		snippetEditor.file.value = "";
	});

	validate();

	return snippetEditor;
}

export function editTimelineSnippet(timelineItemEl: HTMLElement, config: SnippetEditorConfig) {
	// HACK(ben): We just modify the data we got from the server to masquerade as
	// a different user.
	config.owner.name = timelineItemEl.querySelector(".user")!.textContent;
	config.owner.profileUrl = timelineItemEl.querySelector<HTMLAnchorElement>(".user")!.href;
	config.owner.avatarUrl = timelineItemEl.querySelector<HTMLImageElement>(".avatar")?.src;

	const creationDate = new Date(must(timelineItemEl.querySelector<HTMLTimeElement>("time")).dateTime);
	const rawDesc = must(timelineItemEl.querySelector<HTMLElement>(".rawdesc")).textContent;
	const projectIDs: number[] = [];
	const projectEls = timelineItemEl.querySelectorAll<HTMLInputElement>(".project-id-list > input");
	for (const projectEl of projectEls) {
		let projID = parseInt(projectEl.value, 10);
		if (projID) {
			projectIDs.push(projID);
		}
	}
	const snippetEdit = makeSnippetEdit({
		config,
		edit: {
			el: timelineItemEl,
			id: must(timelineItemEl.getAttribute("data-id")),
			creationDate: creationDate,
			text: rawDesc,
			projectIDs,
		},
	});
	timelineItemEl.parentElement!.insertBefore(snippetEdit.root, timelineItemEl);
	timelineItemEl.remove();
}
