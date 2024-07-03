function ImageSelector(form, maxFileSize, container, {
	defaultImageUrl = "",
	onUpdate = (url) => {},
} = {}) {
	this.form = form;
	this.maxFileSize = maxFileSize;
	this.fileInput = container.querySelector(".imginput");
	this.removeImageInput = container.querySelector(".imginput-remove");
	this.imageEl = container.querySelector("img");
	this.container = container.querySelector(".imginput-container");
	this.resetLink = container.querySelector(".imginput-reset-link");
	this.removeLink = container.querySelector(".imginput-remove-link");
	this.filenameText = container.querySelector(".imginput-filename");
	this.originalImageUrl = this.imageEl.getAttribute("data-imginput-original");
	this.originalImageFilename = this.imageEl.getAttribute("data-imginput-original-filename");
	this.currentImageUrl = this.originalImageUrl;
	this.defaultImageUrl = defaultImageUrl;
	this.onUpdate = onUpdate;

	this.fileInput.value = "";
	this.removeImageInput.value = "";

	this.setImageUrl(this.originalImageUrl, true);
	this.updatePreview();

	this.fileInput.addEventListener("change", function(ev) {
		if (this.fileInput.files.length > 0) {
			this.handleNewImageFile(this.fileInput.files[0]);
		}
	}.bind(this));

	this.resetLink.addEventListener("click", function(ev) {
		this.resetImage();
	}.bind(this));

	if (this.removeLink) {
		this.removeLink.addEventListener("click", function(ev) {
			this.removeImage();
		}.bind(this));
	}
}

ImageSelector.prototype.openFileInput = function() {
	this.fileInput.click();
}

ImageSelector.prototype.handleNewImageFile = function(file) {
	if (file) {
		this.updateSizeLimit(file.size);
		this.removeImageInput.value = "";
		this.setImageUrl(URL.createObjectURL(file));
		this.updatePreview(file);
	}
};

ImageSelector.prototype.removeImage = function() {
	this.updateSizeLimit(0);
	this.fileInput.value = "";
	this.removeImageInput.value = "true";
	this.setImageUrl(this.defaultImageUrl);
	this.updatePreview(null);
};

ImageSelector.prototype.resetImage = function() {
	this.updateSizeLimit(0);
	this.fileInput.value = "";
	this.removeImageInput.value = "";
	this.setImageUrl(this.originalImageUrl);
	this.updatePreview(null);
};

ImageSelector.prototype.updateSizeLimit = function(size) {
	this.fileTooBig = size > this.maxFileSize;
	if (this.fileTooBig) {
		this.setError("File too big. Max filesize is " + this.maxFileSize + " bytes.");
	} else {
		this.setError("");
	}
};

ImageSelector.prototype.setError = function(error) {
	this.fileInput.setCustomValidity(error);
	this.fileInput.reportValidity();
}

ImageSelector.prototype.setImageUrl = function(url, initial = false) {
	this.currentImageUrl = url;
	this.imageEl.src = url;
	if (url.length > 0) {
		this.imageEl.style.display = "block";
	} else {
		this.imageEl.style.display = "none";
	}
	this.url = url;
	if (!initial) {
		this.onUpdate(url);
	}
};

ImageSelector.prototype.updatePreview = function(file) {
	const showReset = (
		this.originalImageUrl
		&& this.originalImageUrl != this.defaultImageUrl
		&& this.originalImageUrl != this.currentImageUrl
	);
	const showRemove = (
		!this.fileInput.required
		&& this.currentImageUrl != this.defaultImageUrl
	);
	this.resetLink.hidden = !showReset;
	this.removeLink.hidden = !showRemove;

	if (this.currentImageUrl == this.originalImageUrl) {
		this.filenameText.innerText = this.originalImageFilename;
	} else {
		this.filenameText.innerText = file ? file.name : "";
	}

	this.container.hidden = !this.currentImageUrl;
};
