import { emptyElement, makeTemplateCloner } from "./lib/templates";
import { HTMLFileInputElement } from "./lib/types";
import { assert, must } from "./lib/utils";

const snippetEditTemplate = makeTemplateCloner<{
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

type AvailableProject = {
	id: number,
	name: string,
	logo: string,
};

type SnippetEditOptions = {
	maxFilesize: number,
	availableProjects: AvailableProject[],
	ownerName: string | undefined,
	ownerAvatar: string | undefined,
	ownerUrl: string | undefined,
	date: Date,
	text: string,
	attachmentElement: Element | undefined,
	projectIds: number[],
	stickyProjectId: number | undefined,
	onDeleteRedirectUrl: string | undefined,
	snippetId: string | undefined,
	originalSnippetEl: HTMLElement | undefined,
};

export function makeSnippetEdit({
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
	originalSnippetEl,
}: SnippetEditOptions) {
	const snippetEdit = snippetEditTemplate();
	let projectSelector: HTMLSelectElement | null = null;
	let originalAttachment: Element | null = null;
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
	snippetEdit.username.textContent = ownerName ?? "";
	snippetEdit.username.href = ownerUrl ?? "";
	snippetEdit.date.textContent = new Intl.DateTimeFormat([], { month: "2-digit", day: "2-digit", year: "numeric" }).format(date);
	snippetEdit.text.value = text;
	if (attachmentElement) {
		originalAttachment = attachmentElement.cloneNode(true) as Element;
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
		snippetEdit.cancelLink.addEventListener("click", function () {
			cancel();
		});
	} else {
		snippetEdit.cancelLink.remove();
	}

	function cancel() {
		if (originalSnippetEl) {
			snippetEdit.root.parentElement!.insertBefore(originalSnippetEl, snippetEdit.root);
		}
		snippetEdit.root.remove();
	}

	function addProject(proj: AvailableProject) {
		let projEl = snippetEditProjectTemplate();
		projEl.projectId.value = `${proj.id}`;
		projEl.projectLogo.src = proj.logo;
		projEl.projectName.textContent = proj.name;
		if (proj.id == stickyProjectId) {
			projEl.removeButton.remove();
		} else {
			projEl.removeButton.addEventListener("click", function (ev) {
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
		let projInputs = snippetEdit.projectList.querySelectorAll<HTMLInputElement>("input[name=project_id]");
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
				const option = document.createElement("option");
				option.value = `${remainingProjects[i].id}`;
				option.selected = false;
				option.textContent = remainingProjects[i].name;
				projectSelector.appendChild(option);
			}
			projectSelector.addEventListener("change", ev => {
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

	function setFile(file: File) {
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

	function clearAttachment(restoreOriginal: boolean) {
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

	function setPreview(el: Element | null) {
		if (el) {
			snippetEdit.uploadBox.style.display = "none";
			snippetEdit.previewBox.style.display = "block";
			snippetEdit.uploadResetBox.style.display = "none";
			snippetEdit.previewContent = emptyElement(snippetEdit.previewContent);
			snippetEdit.previewContent.appendChild(el);
			snippetEdit.resetLink.style.display = (!originalAttachment || el == originalAttachment) ? "none" : "inline-block";
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
			// NOTE(asaf): Writing this out in bytes to make the limit exactly clear to the user.
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

	snippetEdit.root.addEventListener("dragover", ev => {
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

	snippetEdit.root.addEventListener("dragenter", ev => {
		assert(ev.dataTransfer);
		enterCounter++;
		const droppable = Array.from(ev.dataTransfer.items).some(
			item => item.kind.toLowerCase() === "file"
		);
		if (droppable) {
			snippetEdit.root.classList.add("drop");
		}
	});

	snippetEdit.root.addEventListener("dragleave", ev => {
		enterCounter--;
		if (enterCounter == 0) {
			snippetEdit.root.classList.remove("drop");
		}
	});

	snippetEdit.root.addEventListener("drop", ev => {
		enterCounter = 0;
		snippetEdit.root.classList.remove("drop");

		if (ev.dataTransfer && ev.dataTransfer.files && ev.dataTransfer.files.length > 0) {
			setFile(ev.dataTransfer.files[0]);
		}

		ev.preventDefault();
	});

	snippetEdit.text.addEventListener("paste", ev => {
		assert(ev.clipboardData);
		const files = ev.clipboardData.files ?? [];
		if (files.length > 0) {
			setFile(files[0]);
		}
	});

	snippetEdit.text.addEventListener("input", () => {
		validate();
	});

	snippetEdit.saveButton.addEventListener("click", ev => {
		let projectsChanged = false;
		let projInputs = snippetEdit.projectList.querySelectorAll<HTMLInputElement>("input[name=project_id]");
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

	snippetEdit.deleteButton.addEventListener("click", function (ev) {
		if (!window.confirm("Are you sure you want to delete this snippet?")) {
			ev.preventDefault();
			return;
		}

		snippetEdit.redirect.value = onDeleteRedirectUrl ?? "";
		snippetEdit.file.value = "";
	});

	validate();

	return snippetEdit;
}

export type EditTimelineSnippetOptions = {
	maxFilesize: number,
	availableProjects: AvailableProject[],
	stickyProjectId?: number,
	onDeleteRedirectUrl?: string,
};

export function editTimelineSnippet(timelineItemEl: HTMLElement, {
	maxFilesize,
	availableProjects,
	stickyProjectId,
	onDeleteRedirectUrl,
}: EditTimelineSnippetOptions) {
	// TODO(ben): must/types
	const ownerName = timelineItemEl.querySelector(".user")?.textContent;
	const ownerUrl = timelineItemEl.querySelector<HTMLAnchorElement>(".user")?.href;
	const ownerAvatar = timelineItemEl.querySelector<HTMLImageElement>(".avatar")?.src;
	const creationDate = new Date(must(timelineItemEl.querySelector<HTMLTimeElement>("time")).dateTime);
	const rawDesc = must(timelineItemEl.querySelector<HTMLElement>(".rawdesc")).textContent;
	const attachment = timelineItemEl.querySelector<HTMLElement>(".timeline-media")?.children?.[0];
	const projectIds: number[] = [];
	const projectEls = timelineItemEl.querySelectorAll<HTMLInputElement>(".project-id-list > input");
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
		originalSnippetEl: timelineItemEl,
	});
	timelineItemEl.parentElement!.insertBefore(snippetEdit.root, timelineItemEl);
	timelineItemEl.remove();
}
