import { ImageSelector } from "./lib/image_selector";
import { setupMarkdownUpload } from "./lib/markdown_upload";
import { getLinkData, initLinkEditor } from "./lib/link_editor";
import { initHashTabs } from "./lib/tabs";
import { makeTemplateCloner } from "./lib/templates";

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
}) {
	initHashTabs(document);

	let projectForm = document.querySelector("#project_form");

	//////////
	// Tags //
	//////////

	const tag = document.querySelector('#tag');
	const tagPreview = document.querySelector('#tag-preview');
	function updateTagPreview() {
		tagPreview.innerText = tag.value || "[your tag]";
	}
	updateTagPreview();
	tag.addEventListener('input', () => updateTagPreview());

	////////////////////////////
	// Description management //
	////////////////////////////

	const description = document.querySelector('#full_description');
	const descPreview = document.querySelector('#desc_preview');
	const { clear: clearDescription } = autosaveContent({
		inputEl: description,
		storageKey: `project-description/${projectName}`,
	});
	projectForm.addEventListener('submit', () => clearDescription());

	// TODO(ben): Probably the live markdown stuff should be module-ified too.
	let doMarkdown = initLiveMarkdown({ inputEl: description, previewEl: descPreview });

	//////////////////////
	// Owner management //
	//////////////////////

	const OWNER_QUERY_STATE_IDLE = 0;
	const OWNER_QUERY_STATE_QUERYING = 1;

	let ownerQueryState = OWNER_QUERY_STATE_IDLE;
	let addOwnerInput = document.querySelector("#owner_name");
	let addOwnerButton = document.querySelector("#owner_add");
	let ownersError = document.querySelector("#owners_error");
	let ownerList = document.querySelector("#owner_list");
	let ownerTemplate = makeTemplateCloner("owner_row");
	let ownerPreviewTemplate = makeTemplateCloner("owner_preview");
	let ownersPreviewContainer = document.querySelector("#owners_preview");

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

	function setOwnerQueryState(state) {
		ownerQueryState = state;
		querying = (ownerQueryState == OWNER_QUERY_STATE_QUERYING);
		addOwnerInput.disabled = querying;
		addOwnerButton.disabled = querying;
		updateAddOwnerStyles();
	}

	function addOwner(username, bestName, avatarUrl) {
		let ownerEl = ownerTemplate();
		ownerEl.input.value = username;
		ownerEl.name.textContent = bestName;
		ownerEl.title = username;
		ownerEl.avatar.src = avatarUrl;
		ownerList.appendChild(ownerEl.root);
		updateAddOwnerStyles();
		updateOwnersPreview();
	}

	ownerList.addEventListener("click", function (ev) {
		if (ev.target.closest(".remove_owner")) {
			ev.target.closest(".owner_row").remove();
		}
		updateAddOwnerStyles();
		updateOwnersPreview();
	});

	function updateOwnersPreview() {
		let ownerEls = ownerList.querySelectorAll(".owner_row");
		ownersPreviewContainer.innerHTML = "";
		for (let i = 0; i < ownerEls.length; ++i) {
			let avatarUrl = ownerEls[i].querySelector("img").src;
			let name = ownerEls[i].querySelector("span").textContent;
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
	function openLogoSelector(e) {
		e.preventDefault();
		logoSelector.openImageInput();
	}
	document.querySelector("#project-logo-placeholder").replaceWith(logoSelector.root);

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
	function openHeaderSelector(e) {
		e.preventDefault();
		headerSelector.openImageInput();
	}
	document.querySelector("#header-image-placeholder").replaceWith(headerSelector.root);

	function updateCardPreview() {
		const title = document.querySelector("#project_name").value || "Project Title";

		document.querySelector("#logo_preview img").src = logoSelector.url;
		document.querySelector("#logo_placeholder").innerText = title[0].toUpperCase();
		document.querySelector("#logo_placeholder").hidden = !!logoSelector.url;
		document.querySelector("#header_img_preview").style.backgroundImage = `url(${headerSelector.url})`;
		document.querySelector("#flowsnake").classList.toggle("dn", headerSelector.url);
		document.querySelector("#name_preview").innerText = title;
		document.querySelector("#longdesc_title").innerText = title;
		document.querySelector("#blurb_preview").innerText = document.querySelector("#description").value || "Project summary";
	}
	updateCardPreview();

	//////////////////
	// Asset upload //
	//////////////////
	setupMarkdownUpload(
		document.querySelectorAll("#project_form input[type=submit]"),
		document.querySelector('#file_input'),
		document.querySelector('.upload_bar'),
		description,
		doMarkdown,
		textMaxFileSize,
		editorUploadUrl,
	);

	/////////////////////
	// Link management //
	/////////////////////

	initLinkEditor(initialLinks);

	const primaryLinkTemplate = makeTemplateCloner("primary_link");
	const secondaryLinkTemplate = makeTemplateCloner("secondary_link");

	function updateLinkPreviews() {
		const linkData = getLinkData();

		const primaryPreview = document.querySelector("#primary_links_preview");
		const secondaryPreview = document.querySelector("#secondary_links_preview");

		primaryPreview.innerHTML = "";
		secondaryPreview.innerHTML = "";

		for (const link of linkData) {
			if (link.primary) {
				const l = primaryLinkTemplate();
				l.root.href = link.url;
				l.name.innerText = link.name;
				primaryPreview.appendChild(l.root);
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
				const iconSVG = document.querySelector(`#link-icon-${icon}`).innerHTML;

				const l = secondaryLinkTemplate();
				l.root.href = link.url;
				l.root.title = link.name || title;
				l.root.innerHTML = iconSVG;
				secondaryPreview.appendChild(l.root);
			}
		}
	}
	updateLinkPreviews();
	window.addEventListener("linkedit", () => updateLinkPreviews());
}
