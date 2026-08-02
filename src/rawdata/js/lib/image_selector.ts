import { makeTemplateCloner } from "./templates";

export type ImageSelectorOptions = {
	defaultUrl?: string,
	original?: OriginalFile,

	onUpdate?: ImageSelectorUpdateFunc,
	onRemove?: () => void,
};
export type ImageSelectorUpdateFunc = (url: string) => void;

// NOTE(ben): Fields chosen to line up with Asset.
type OriginalFile = {
	url: string,
	filename: string,
	id: string,
};

// NOTE(ben): See image_selector.html.
type ImageSelectorTmpl = {
	inputOriginal: HTMLInputElement,
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
	private originalImageInput: HTMLInputElement;
	private newImageInput: HTMLInputElement;
	private removeImageInput: HTMLInputElement;
	private previewImage: HTMLImageElement;
	private previewContainer: HTMLElement;
	private resetLink: HTMLAnchorElement;
	private removeLink: HTMLAnchorElement;
	private filenameText: HTMLElement;
	private errorEl: HTMLElement;
	private defaultUrl: string;
	private originalFile: OriginalFile | null;
	private onUpdate: ImageSelectorUpdateFunc;
	private onRemove: () => void;

	constructor(
		formName: string,
		maxFileSize: number,
		{
			defaultUrl = "",
			original,

			onUpdate = () => { },
			onRemove = () => { },
		}: ImageSelectorOptions = {},
	) {
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

		// NOTE(ben): Initialize DOM things
		this.originalImageInput.name = `original_${formName}`;
		this.originalImageInput.value = original?.id ?? "NOASSET";

		this.newImageInput.name = `image_${formName}`;
		this.newImageInput.value = "";

		this.removeImageInput.name = `remove_${formName}`;
		this.removeImageInput.value = "";

		this.setImageUrl(this.originalFile?.url ?? "", /*initial=*/true);
		this.updatePreview();

		this.newImageInput.addEventListener("change", ev => {
			if (this.newImageInput.files!.length > 0) {
				this.handleNewImageFile(this.newImageInput.files![0]);
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
				if (this.newImageInput.files!.length > 0) {
					resolve(this.newImageInput.files![0]);
				} else {
					resolve(null);
				}
			}
			this.newImageInput.addEventListener("change", done, { once: true });
			this.newImageInput.addEventListener("cancel", done, { once: true });
			this.newImageInput.click();
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
		this.newImageInput.setCustomValidity(error);
		this.newImageInput.reportValidity();
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
			this.originalFile
			&& this.originalFile.url !== this.defaultUrl
			&& this.originalFile.url !== this.url
		);
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

	private handleNewImageFile(file: File) {
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
}
