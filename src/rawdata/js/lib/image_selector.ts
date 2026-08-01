import { makeTemplateCloner } from "./templates";

export type ImageSelectorOptions = {
	defaultUrl?: string,
	originalUrl?: string,
	originalFilename?: string,

	onUpdate?: ImageSelectorUpdateFunc,
	onRemove?: () => void,
};
export type ImageSelectorUpdateFunc = (url: string) => void;

// NOTE(ben): See image_selector.html.
type ImageSelectorTmpl = {
	inputImage: HTMLInputElement,
	inputRemove: HTMLInputElement,
	errorMessage: HTMLElement,
	previewContainer: HTMLElement,
	preview: HTMLImageElement,
	linkReset: HTMLAnchorElement,
	linkRemove: HTMLAnchorElement,
	filenameText: HTMLElement,
}
const imageSelectorTemplate = makeTemplateCloner<ImageSelectorTmpl>("image-selector");

export class ImageSelector {
	// The current URL of the selector. (Will be an object URL if the image has
	// not been submitted yet.)
	url: string;

	// The template content to be inserted into the DOM wherever you like.
	root: DocumentFragment;

	// 

	private maxFileSize: number;
	private imageInput: HTMLInputElement;
	private removeImageInput: HTMLInputElement;
	private previewImage: HTMLImageElement;
	private previewContainer: HTMLElement;
	private resetLink: HTMLAnchorElement;
	private removeLink: HTMLAnchorElement;
	private filenameText: HTMLElement;
	private errorEl: HTMLElement;
	private defaultUrl: string;
	private originalUrl: string;
	private originalFilename: string;
	private onUpdate: ImageSelectorUpdateFunc;
	private onRemove: () => void;

	constructor(
		formName: string,
		maxFileSize: number,
		{
			defaultUrl = "",
			originalUrl = "",
			originalFilename = "",

			onUpdate = () => { },
			onRemove = () => { },
		}: ImageSelectorOptions = {},
	) {
		const tmpl = imageSelectorTemplate();

		this.url = "";
		this.root = tmpl.root;

		this.maxFileSize = maxFileSize;

		this.imageInput = tmpl.inputImage;
		this.removeImageInput = tmpl.inputRemove;
		this.previewImage = tmpl.preview;
		this.previewContainer = tmpl.previewContainer;
		this.resetLink = tmpl.linkReset;
		this.removeLink = tmpl.linkRemove;
		this.filenameText = tmpl.filenameText;
		this.errorEl = tmpl.errorMessage;

		this.defaultUrl = defaultUrl;
		this.originalUrl = originalUrl;
		this.originalFilename = originalFilename;

		this.onUpdate = onUpdate;
		this.onRemove = onRemove;

		// NOTE(ben): Initialize DOM things
		this.imageInput.name = formName;
		this.imageInput.value = "";

		this.removeImageInput.name = `remove_${formName}`;
		this.removeImageInput.value = "";

		this.setImageUrl(this.originalUrl, /*initial=*/true);
		this.updatePreview();

		this.imageInput.addEventListener("change", ev => {
			if (this.imageInput.files!.length > 0) {
				this.handleNewImageFile(this.imageInput.files![0]);
			}
		});

		this.resetLink.addEventListener("click", ev => {
			this.resetImage();
		});

		if (this.removeLink) {
			this.removeLink.addEventListener("click", ev => {
				this.removeImage();
			});
		}
	}

	openImageInput(): Promise<File | null> {
		return new Promise(resolve => {
			const done = () => {
				if (this.imageInput.files!.length > 0) {
					resolve(this.imageInput.files![0]);
				} else {
					resolve(null);
				}
			}
			this.imageInput.addEventListener("change", done, { once: true });
			this.imageInput.addEventListener("cancel", done, { once: true });
			this.imageInput.click();
		});
	}
	setImageUrl(url: string, initial = false) {
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
	private setError(error: string) {
		this.errorEl.textContent = error;
		this.errorEl.hidden = !error;
		this.imageInput.setCustomValidity(error);
		this.imageInput.reportValidity();
	}
	private checkSizeLimit(size: number) {
		if (size > this.maxFileSize) {
			this.setError("File too big. Max file size is " + this.maxFileSize + " bytes.");
		} else {
			this.setError("");
		}
	}
	private updatePreview(file: File | null = null) {
		const showReset = (
			this.originalUrl
			&& this.originalUrl !== this.defaultUrl
			&& this.originalUrl !== this.url
		);
		const showRemove = this.url !== this.defaultUrl;
		this.resetLink.hidden = !showReset;
		this.removeLink.hidden = !showRemove;

		if (this.url === this.originalUrl) {
			this.filenameText.innerText = this.originalFilename;
		} else {
			this.filenameText.innerText = file ? file.name : "";
		}

		this.previewContainer.hidden = !this.url;
	}

	private handleNewImageFile(file: File) {
		this.checkSizeLimit(file.size);
		this.removeImageInput.value = "";
		this.setImageUrl(URL.createObjectURL(file));
		this.updatePreview(file);
	}
	removeImage() {
		this.checkSizeLimit(0);
		this.imageInput.value = "";
		this.removeImageInput.value = "true";
		this.setImageUrl(this.defaultUrl);
		this.updatePreview(null);
		this.onRemove();
	}
	resetImage() {
		this.checkSizeLimit(0);
		this.imageInput.value = "";
		this.removeImageInput.value = "";
		this.setImageUrl(this.originalUrl);
		this.updatePreview(null);
	}
}
