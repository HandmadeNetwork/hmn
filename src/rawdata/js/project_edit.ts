import { ImageSelector } from "./lib/image_selector";
import { setupMarkdownUpload } from "./lib/markdown_upload";
import { Asset, Icon } from "./lib/models";
import { getLinkData, initLinkEditor, LinkData } from "./lib/link_editor";
import { initHashTabs } from "./lib/tabs";
import { makeTemplateCloner } from "./lib/templates";
import { CSRFToken, HTMLFileInputElement } from "./lib/types";
import { must } from "./lib/utils";
import { autosaveContent, initLiveMarkdown } from "./lib/markdown_previews";
import { initReorderable } from "./lib/reorderable";
import { CheckUsernameResult } from "./lib/apitypes";
import { updateStickySidebars } from "./lib/sticky_sidebar";
import "./lib/relocator";
import { firstInvalidElement, FormElement } from "./lib/validation";

export type ProjectEditConfig = {
	csrf: CSRFToken,
	projectName: string,
	maxOwners: number,
	maxScreenshots: number,
	logoMaxFileSize: number,
	screenshotMaxFileSize: number,
	textMaxFileSize: number,
	editorUploadUrl: string,
	initialLinks: LinkData,
	ownerCheckUrl: string,
	logo: Asset | null,
	screenshots: Asset[] | null,
	linkIcons: Icon[],
};

export function init({
	csrf,
	projectName,
	maxOwners,
	maxScreenshots,
	logoMaxFileSize,
	screenshotMaxFileSize,
	textMaxFileSize,
	editorUploadUrl,
	initialLinks,
	ownerCheckUrl,
	logo,
	screenshots,
	linkIcons: linkIconsRaw,
}: ProjectEditConfig) {
	const projectForm = must(document.querySelector<HTMLFormElement>("#project_form"));

	//////////
	// Tabs //
	//////////
	const { selectTab } = initHashTabs(document, {
		onSelect(name) {
			const card = must(document.querySelector<HTMLElement>("#card-preview-sticky-container"));
			const description = must(document.querySelector<HTMLElement>("#description-preview-sticky-container"));
			const links = must(document.querySelector<HTMLElement>("#links-preview-sticky-container"));

			card.hidden = name !== "info";
			description.hidden = name !== "info";
			links.hidden = name !== "images";

			return true;
		},
		fireOnSelectForInitialTab: true,
	});
	const previewResizeObserver = new ResizeObserver(updateStickySidebars);
	previewResizeObserver.observe(must(document.querySelector("#preview-container")));

	////////////////
	// Validation //
	////////////////
	projectForm.addEventListener("submit", e => {
		const firstBadElement = firstInvalidElement(projectForm);
		if (!firstBadElement) {
			return;
		}

		// NOTE(ben): We stop immediate propagation to prevent other submit
		// handlers from firing, e.g. to clear saved markdown.
		e.preventDefault();
		e.stopImmediatePropagation();

		const tab = must(firstBadElement.closest("[data-tab]"));
		selectTab(tab.getAttribute("data-tab")!);
		firstBadElement.focus();
		firstBadElement.reportValidity(); // NOTE(ben): Without this the browser won't show the validation popup.

		// MUSING(ben): The browser popup is ugly and stupid, but I doubt it's
		// worth building custom validation popups either. The integration with the
		// built-in browser validation could maybe be managed with the info present
		// in ValidityState, but unless the browser provides us with a complete
		// string to present, localization concerns would be tremendously annoying.
	});

	//////////
	// Tags //
	//////////
	{
		const tag = must(document.querySelector<HTMLInputElement>('#tag'));
		const tagPreview = must(document.querySelector<HTMLElement>('#tag-preview'));
		function updateTagPreview() {
			tagPreview.innerText = tag.value || "[your tag]";
		}
		updateTagPreview();
		tag.addEventListener('input', () => updateTagPreview());
	}

	////////////////////////////
	// Description management //
	////////////////////////////
	{
		const description = must(document.querySelector<HTMLTextAreaElement>('#full_description'));
		const aiPolicy = must(document.querySelector<HTMLTextAreaElement>('#ai_policy'));
		const descPreview = must(document.querySelector<HTMLElement>('#desc_preview'));
		const aiPolicyPreview = must(document.querySelector<HTMLElement>('#ai_policy_preview'));
		const { clear: clearDescription } = autosaveContent({
			inputEl: description,
			storageKey: `project-description/${projectName}`,
		});
		projectForm.addEventListener('submit', () => clearDescription());
		previewResizeObserver.observe(description);

		const descMarkdown = initLiveMarkdown({ inputEl: description, previewEl: descPreview });
		const aiPolicyMarkdown = initLiveMarkdown({ inputEl: aiPolicy, previewEl: aiPolicyPreview });
		setupMarkdownUpload(
			document.querySelectorAll("#project_form input[type=submit]"),
			must(document.querySelector<HTMLFileInputElement>('#file_input')),
			must(document.querySelector('.upload_bar')),
			description,
			descMarkdown,
			textMaxFileSize,
			editorUploadUrl,
		);
	}

	//////////////////////
	// Owner management //
	//////////////////////
	initOwnersUI({ csrf, maxOwners, ownerCheckUrl });

	//////////////////////////////
	// Logo / header management //
	//////////////////////////////
	{
		const projectNameField = must(document.querySelector<HTMLInputElement>("#project_name"));
		const descriptionField = must(document.querySelector<HTMLTextAreaElement>("#description"));

		const logoSelector = new ImageSelector(
			"logo",
			logoMaxFileSize,
			{
				original: logo || undefined,
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

		function updateCardPreview() {
			const title = projectNameField.value || "Project Title";

			must(document.querySelector<HTMLImageElement>("#logo_preview img")).src = logoSelector.url;
			must(document.querySelector<HTMLElement>("#logo_placeholder")).innerText = title[0].toUpperCase();
			must(document.querySelector<HTMLElement>("#logo_placeholder")).hidden = !!logoSelector.url;
			must(document.querySelector<HTMLElement>("#name_preview")).innerText = title;
			must(document.querySelector<HTMLElement>("#blurb_preview")).innerText = descriptionField.value || "Project summary";

			// NOTE(ben): Also update guidance in the editor.
			must(document.querySelector<HTMLElement>("#logo-selector-container")).hidden = !logoSelector.url;
		}
		updateCardPreview();
		projectNameField.addEventListener("input", updateCardPreview);
		descriptionField.addEventListener("input", updateCardPreview);
	}

	/////////////////////
	// Link management //
	/////////////////////
	{
		initLinkEditor(initialLinks);
		const linkIcons = Object.fromEntries(linkIconsRaw.map(i => [i.name, i.svg]));

		const linkTemplate = makeTemplateCloner<{
			link: HTMLAnchorElement,
			name: HTMLElement,
			icon: HTMLElement,
			divider: HTMLElement,
		}>("link-preview");

		function updateLinkPreviews() {
			const linkData = getLinkData();

			const preview = must(document.querySelector<HTMLElement>("#links-preview"));
			preview.innerHTML = "";

			let seenPrimary = false, seenSecondary = false;
			for (const link of linkData) {
				const name = link.name || link.serviceName;
				const iconSVG = linkIcons[link.icon];

				const l = linkTemplate();
				l.link.href = link.url;
				l.name.innerText = name;
				l.name.title = name;
				l.icon.innerHTML = iconSVG;
				l.divider.hidden = link.primary || !seenPrimary || seenSecondary;
				preview.appendChild(l.root);

				seenPrimary ||= link.primary;
				seenSecondary ||= !link.primary;
			}
		}
		updateLinkPreviews();
		window.addEventListener("wasmready", () => updateLinkPreviews());
		window.addEventListener("linkedit", () => updateLinkPreviews());
	}

	///////////////////////////
	// Screenshot management //
	///////////////////////////
	{
		const screenshotContainer = must(document.querySelector<HTMLElement>("#screenshots"));
		const newScreenshotButton = must(document.querySelector<HTMLButtonElement>("#screenshot-upload-button"));
		const screenshotTemplate = makeTemplateCloner<{
			root: HTMLElement,
			grabHandle: HTMLElement,
			selectorPlaceholder: HTMLElement,
		}>("screenshot");

		// Initialize with existing screenshots
		const { startDrag: startDragScreenshot } = initReorderable(screenshotContainer, {
			onReorder() {
				// TODO(ben): Update whatever preview exists
			}
		});
		for (const screenshot of screenshots ?? []) {
			const el = screenshotTemplate();

			const selector = new ImageSelector("screenshot", screenshotMaxFileSize, {
				original: screenshot,
				onRemove: () => {
					el.root.hidden = true;
					updateNewScreenshotButton();
				},
			});
			el.selectorPlaceholder.replaceWith(selector.root);
			el.grabHandle.addEventListener("pointerdown", startDragScreenshot);

			screenshotContainer.appendChild(el.root);
		}

		// Hook up new screenshots
		newScreenshotButton.addEventListener("click", async e => {
			e.preventDefault();
			const el = screenshotTemplate();
			const selector = new ImageSelector("screenshot", screenshotMaxFileSize, {
				onRemove: () => {
					el.root.remove();
					updateNewScreenshotButton();
				}
			});
			el.selectorPlaceholder.replaceWith(selector.root);
			el.grabHandle.addEventListener("pointerdown", startDragScreenshot);

			// NOTE(ben): The element needs to be in the real DOM to actually pop up
			// the dialog, but if we insert it in the right place, styles will break.
			// So, insert it somewhere stupid and then move it.
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

			updateNewScreenshotButton();
		});

		function updateNewScreenshotButton() {
			const numScreenshots = screenshotContainer.querySelectorAll(".reorderable-item:not([hidden])").length;
			newScreenshotButton.disabled = numScreenshots >= maxScreenshots;
		}
		updateNewScreenshotButton();
	}
}

function initOwnersUI(opts: {
	csrf: CSRFToken,
	maxOwners: number,
	ownerCheckUrl: string,
}) {
	type OwnerQueryState = "idle" | "querying";

	let ownerQueryState: OwnerQueryState = "idle";
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

	function updateAddOwnerStyles() {
		const numOwnerRows = ownerList.querySelectorAll('.owner_row').length;
		addOwnerInput.disabled = numOwnerRows >= opts.maxOwners;
	}
	updateAddOwnerStyles();

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

	async function startAddOwner() {
		if (ownerQueryState == "querying") {
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
		setOwnerQueryState("querying");
		try {
			const data = new FormData();
			data.append(opts.csrf.field, opts.csrf.token);
			data.append("username", newOwner);

			const response = await fetch(opts.ownerCheckUrl, {
				method: "POST",
				credentials: "include",
				body: data,
			});
			const result = await response.json() as CheckUsernameResult;
			if (result.found) {
				addOwner(result.username, result.name, result.avatarUrl);
				addOwnerInput.value = "";
			} else {
				ownersError.textContent = "Username not found.";
			}

			if (document.activeElement === addOwnerButton) {
				addOwnerInput.focus();
			}
		} catch (e) {
			console.error(e);
			ownersError.textContent = "There was an issue validating this username";
		}
		setOwnerQueryState("idle");
	}

	function setOwnerQueryState(state: OwnerQueryState) {
		ownerQueryState = state;
		const querying = (ownerQueryState === "querying");
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
		if ((ev.target as Element).closest(".remove_owner")) {
			must((ev.target as Element).closest(".owner_row")).remove();
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
}