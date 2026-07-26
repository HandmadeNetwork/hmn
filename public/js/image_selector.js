// NOTE(ben): See image_selector.html.
const imageSelectorTemplate = makeTemplateCloner("image-selector");

class ImageSelector {
	constructor(
		formName,
		maxFileSize,
		{
			defaultUrl = "",
			originalUrl = "",
			originalFilename = "",

			onUpdate = url => {},
		} = {},
	) {
		const tmpl = imageSelectorTemplate();
		
		// ------------------------------------------------------------------------
		// NOTE(ben): "Public" properties

		// The current URL of the selector. (Will be an object URL if the image has
		// not been submitted yet.)
		this.url = ""; 

		// The template content to be inserted into the DOM wherever you like.
		this.root = tmpl.root;

		// ------------------------------------------------------------------------
		// NOTE(ben): Set other "private" instance variables

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

		// NOTE(ben): Initialize DOM things
		this.imageInput.name = formName;
		this.imageInput.value = "";

		this.removeImageInput.name = `remove_${formName}`;
		this.removeImageInput.value = "";

		this.setImageUrl(this.originalUrl, /*initial=*/true);
		this.updatePreview();

		this.imageInput.addEventListener("change", function (ev) {
			if (this.imageInput.files.length > 0) {
				this.handleNewImageFile(this.imageInput.files[0]);
			}
		}.bind(this));

		this.resetLink.addEventListener("click", function (ev) {
			this.resetImage();
		}.bind(this));

		if (this.removeLink) {
			this.removeLink.addEventListener("click", function (ev) {
				this.removeImage();
			}.bind(this));
		}
	}

	openImageInput() {
		this.imageInput.click();
	}
	setImageUrl(url, initial = false) {
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
	setError(error) {
		this.errorEl.textContent = error;
		this.errorEl.hidden = !error;
		this.imageInput.setCustomValidity(error);
		this.imageInput.reportValidity();
	}
	checkSizeLimit(size) {
		if (size > this.maxFileSize) {
			this.setError("File too big. Max file size is " + this.maxFileSize + " bytes.");
		} else {
			this.setError("");
		}
	}
	updatePreview(file = null) {
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

	handleNewImageFile(file) {
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
	}
	resetImage() {
		this.checkSizeLimit(0);
		this.imageInput.value = "";
		this.removeImageInput.value = "";
		this.setImageUrl(this.originalUrl);
		this.updatePreview(null);
	}
}
