function ImageSelector(form, maxFileSize, container, defaultImageUrl) {
	this.form = form;
	this.maxFileSize = maxFileSize;
	this.fileInput = container.querySelector(".image_input");
	this.removeImageInput = container.querySelector(".remove_input");
	this.imageEl = container.querySelector("img");
	this.resetLink = container.querySelector(".reset");
	this.removeLink = container.querySelector(".remove");
	this.originalImageUrl = this.imageEl.getAttribute("data-original");
	this.currentImageUrl = this.originalImageUrl;
	this.defaultImageUrl = defaultImageUrl || "";

	this.fileInput.value = "";
	this.removeImageInput.value = "";

	this.setImageUrl(this.originalImageUrl);
	this.updateButtons();

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

ImageSelector.prototype.handleNewImageFile = function(file) {
	if (file) {
		this.updateSizeLimit(file.size);
		this.removeImageInput.value = "";
		this.setImageUrl(URL.createObjectURL(file));
		this.updateButtons();
	}
};

ImageSelector.prototype.removeImage = function() {
	this.updateSizeLimit(0);
	this.fileInput.value = "";
	this.removeImageInput.value = "true";
	this.setImageUrl(this.defaultImageUrl);
	this.updateButtons();
};

ImageSelector.prototype.resetImage = function() {
	this.updateSizeLimit(0);
	this.fileInput.value = "";
	this.removeImageInput.value = "";
	this.setImageUrl(this.originalImageUrl);
	this.updateButtons();
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

ImageSelector.prototype.setImageUrl = function(url) {
	this.currentImageUrl = url;
	this.imageEl.src = url;
	if (url.length > 0) {
		this.imageEl.style.display = "block";
	} else {
		this.imageEl.style.display = "none";
	}
};

ImageSelector.prototype.updateButtons = function() {
	if ((this.originalImageUrl.length > 0 && this.originalImageUrl != this.defaultImageUrl)
		&& this.currentImageUrl != this.originalImageUrl) {

		this.resetLink.style.display = "inline-block";
	} else {
		this.resetLink.style.display = "none";
	}

	if (!this.fileInput.required && this.currentImageUrl != this.defaultImageUrl) {
		this.removeLink.style.display = "inline-block";
	} else {
		this.removeLink.style.display = "none";
	}
};

