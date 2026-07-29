import { ImageSelector } from "./lib/image_selector";
import { setupMarkdownUpload } from "./lib/markdown_upload";
import { getLinkData, initLinkEditor, LinkData } from "./lib/link_editor";
import { initHashTabs } from "./lib/tabs";
import { makeTemplateCloner } from "./lib/templates";
import { CSRFToken, FileInputElement } from "./lib/types";
import { must } from "./lib/utils";

export type ProjectEditConfig = {
	csrf: CSRFToken,
	projectName: string,
	maxOwners: number,
	logoMaxFileSize: number,
	headerMaxFileSize: number,
	textMaxFileSize: number,
	editorUploadUrl: string,
	initialLinks: LinkData,
	ownerCheckUrl: string,
	logoUrl: string,
	logoFilename: string,
	headerImageUrl: string,
	headerImageFilename: string,
};

export function init({
	csrf,
	projectName,
	maxOwners,
	logoMaxFileSize,
	headerMaxFileSize,
	textMaxFileSize,
	editorUploadUrl,
	initialLinks,
	ownerCheckUrl,
	logoUrl,
	logoFilename,
	headerImageUrl,
	headerImageFilename,
}: ProjectEditConfig) {
	initHashTabs(document);

	const projectForm = must(document.querySelector<HTMLFormElement>("#project_form"));

	//////////
	// Tags //
	//////////

	const tag = must(document.querySelector<HTMLInputElement>('#tag'));
	const tagPreview = must(document.querySelector<HTMLElement>('#tag-preview'));
	function updateTagPreview() {
		tagPreview.innerText = tag.value || "[your tag]";
	}
	updateTagPreview();
	tag.addEventListener('input', () => updateTagPreview());

	////////////////////////////
	// Description management //
	////////////////////////////

	const description = must(document.querySelector<HTMLTextAreaElement>('#full_description'));
	const descPreview = must(document.querySelector<HTMLElement>('#desc_preview'));
	const { clear: clearDescription } = autosaveContent({
		inputEl: description,
		storageKey: `project-description/${projectName}`,
	});
	projectForm.addEventListener('submit', () => clearDescription());

	// TODO(ben): Probably the live markdown stuff should be module-ified too.
	const doMarkdown = initLiveMarkdown({ inputEl: description, previewEl: descPreview });

	//////////////////////
	// Owner management //
	//////////////////////

	const OWNER_QUERY_STATE_IDLE = 0;
	const OWNER_QUERY_STATE_QUERYING = 1;
	type OwnerQueryState = typeof OWNER_QUERY_STATE_IDLE | typeof OWNER_QUERY_STATE_QUERYING;

	let ownerQueryState: OwnerQueryState = OWNER_QUERY_STATE_IDLE;
	const addOwnerInput = must(document.querySelector<HTMLInputElement>("#owner_name"));
	const addOwnerButton = must(document.querySelector<HTMLButtonElement>("#owner_add"));
	const ownersError = must(document.querySelector<HTMLElement>("#owners_error"));
	const ownerList = must(document.querySelector<HTMLElement>("#owner_list"));
	const ownerTemplate = makeTemplateCloner<{
		rootElement: HTMLElement,
		input: HTMLInputElement,
		avatar: HTMLImageElement,
		name: HTMLElement,
	}>("owner_row");
	const ownerPreviewTemplate = makeTemplateCloner<{
		avatar: HTMLImageElement,
		name: HTMLElement,
	}>("owner_preview");
	const ownersPreviewContainer = must(document.querySelector<HTMLElement>("#owners_preview"));

	addOwnerInput.addEventListener("keypress", function (ev) {
		if (ev.which == 13) {
			startAddOwner();
			ev.preventDefault();
			ev.stopPropagation();
		}
	});

	addOwnerButton.addEventListener("click", function (ev) {
		ev.preventDefault();
		startAddOwner();
	});

	function updateAddOwnerStyles() {
		const numOwnerRows = document.querySelectorAll('.owner_row').length;
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
		let ownerEls = ownerList.querySelectorAll<HTMLInputElement>(".owner_row input[name='owners']");
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
		xhr.addEventListener("load", function (ev) {
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
		xhr.addEventListener("error", function (ev) {
			ownersError.textContent = "There was an issue validating this username";
			setOwnerQueryState(OWNER_QUERY_STATE_IDLE);
		});
		let data = new FormData();
		data.append(csrf.field, csrf.token);
		data.append("username", newOwner);
		xhr.send(data);
		setOwnerQueryState(OWNER_QUERY_STATE_QUERYING);
	}

	function setOwnerQueryState(state: OwnerQueryState) {
		ownerQueryState = state;
		const querying = (ownerQueryState == OWNER_QUERY_STATE_QUERYING);
		addOwnerInput.disabled = querying;
		addOwnerButton.disabled = querying;
		updateAddOwnerStyles();
	}

	function addOwner(username: string, bestName: string, avatarUrl: string) {
		let ownerEl = ownerTemplate();
		ownerEl.input.value = username;
		ownerEl.name.textContent = bestName;
		ownerEl.avatar.src = avatarUrl;
		ownerList.appendChild(ownerEl.rootElement);
		updateAddOwnerStyles();
		updateOwnersPreview();
	}

	ownerList.addEventListener("click", ev => {
		if (ownerList.closest(".remove_owner")) {
			must(ownerList.closest(".owner_row")).remove();
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

	//////////////////////////////
	// Logo / header management //
	//////////////////////////////

	const logoSelector = new ImageSelector(
		"logo",
		logoMaxFileSize,
		{
			originalUrl: logoUrl,
			originalFilename: logoFilename,
			onUpdate() {
				updateCardPreview();
			},
		},
	);
	must(document.querySelector("#logo-upload-button")).addEventListener("click", e => {
		e.preventDefault();
		logoSelector.openImageInput();
	});
	must(document.querySelector("#project-logo-placeholder")).replaceWith(logoSelector.root);

	const headerSelector = new ImageSelector(
		"header_image",
		headerMaxFileSize,
		{
			originalUrl: headerImageUrl,
			originalFilename: headerImageFilename,
			onUpdate() {
				updateCardPreview();
			},
		},
	);
	must(document.querySelector("#header-upload-button")).addEventListener("click", e => {
		e.preventDefault();
		headerSelector.openImageInput();
	});
	must(document.querySelector("#header-image-placeholder")).replaceWith(headerSelector.root);

	function updateCardPreview() {
		const title = must(document.querySelector<HTMLInputElement>("#project_name")).value || "Project Title";

		must(document.querySelector<HTMLImageElement>("#logo_preview img")).src = logoSelector.url;
		must(document.querySelector<HTMLElement>("#logo_placeholder")).innerText = title[0].toUpperCase();
		must(document.querySelector<HTMLElement>("#logo_placeholder")).hidden = !!logoSelector.url;
		must(document.querySelector<HTMLElement>("#header_img_preview")).style.backgroundImage = `url(${headerSelector.url})`;
		must(document.querySelector<HTMLElement>("#flowsnake")).classList.toggle("dn", !!headerSelector.url);
		must(document.querySelector<HTMLElement>("#name_preview")).innerText = title;
		must(document.querySelector<HTMLElement>("#longdesc_title")).innerText = title;
		must(document.querySelector<HTMLElement>("#blurb_preview")).innerText = must(document.querySelector<HTMLTextAreaElement>("#description")).value || "Project summary";
	}
	updateCardPreview();

	//////////////////
	// Asset upload //
	//////////////////
	setupMarkdownUpload(
		document.querySelectorAll("#project_form input[type=submit]"),
		must(document.querySelector<FileInputElement>('#file_input')),
		must(document.querySelector('.upload_bar')),
		description,
		doMarkdown,
		textMaxFileSize,
		editorUploadUrl,
	);

	/////////////////////
	// Link management //
	/////////////////////

	initLinkEditor(initialLinks);

	const primaryLinkTemplate = makeTemplateCloner<{
		link: HTMLAnchorElement,
		name: HTMLElement,
	}>("primary_link");
	const secondaryLinkTemplate = makeTemplateCloner<{
		link: HTMLAnchorElement,
	}>("secondary_link");

	function updateLinkPreviews() {
		const linkData = getLinkData();

		const primaryPreview = must(document.querySelector<HTMLElement>("#primary_links_preview"));
		const secondaryPreview = must(document.querySelector<HTMLElement>("#secondary_links_preview"));

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
				// TODO(ben): Functions defined in Go code (.d.ts?)
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
	window.addEventListener("linkedit", () => updateLinkPreviews());
}
