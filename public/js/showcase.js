const TimelineMediaTypes = {
    IMAGE: 1,
    VIDEO: 2,
    AUDIO: 3,
    EMBED: 4,
}

const showcaseItemTemplate = makeTemplateCloner("showcase_item");
const modalTemplate = makeTemplateCloner("timeline_modal");

function showcaseTimestamp(rawDate) {
    const date = new Date(rawDate*1000);
    return date.toLocaleDateString([], { 'dateStyle': 'long' });
}

function doOnce(f) {
    let did = false;
    return () => {
        if (!did) {
            f();
            did = true;
        }
    }
}

function makeShowcaseItem(timelineItem) {
    const timestamp = showcaseTimestamp(timelineItem.date);

    const itemEl = showcaseItemTemplate();
    itemEl.avatar.style.backgroundImage = `url('${timelineItem.owner_avatar}')`;
    itemEl.username.textContent = timelineItem.owner_name;
    itemEl.when.textContent = timestamp;

    let addThumbnailFunc = () => {};
    let createModalContentFunc = () => {};

    switch (timelineItem.media_type) {
    case TimelineMediaTypes.IMAGE:
        addThumbnailFunc = () => {
            itemEl.thumbnail.style.backgroundImage = `url('${timelineItem.thumbnail_url}')`;
        };
        
        createModalContentFunc = () => {
            const modalImage = document.createElement('img');
            modalImage.src = timelineItem.asset_url;
            modalImage.classList.add('mw-100', 'mh-60vh');
            return modalImage;
        };

        break;
    case TimelineMediaTypes.VIDEO:
        addThumbnailFunc = () => {
            const video = document.createElement('video');
            video.src = timelineItem.asset_url; // TODO: Use image thumbnails
            video.controls = false;
            video.classList.add('h-100');
            video.preload = 'metadata';
            itemEl.thumbnail.appendChild(video);
        };

        createModalContentFunc = () => {
            const modalVideo = document.createElement('video');
            modalVideo.src = timelineItem.asset_url;
            modalVideo.controls = true;
            modalVideo.preload = 'metadata';
            modalVideo.classList.add('mw-100', 'mh-60vh');
            return modalVideo;
        };

        break;
    case TimelineMediaTypes.AUDIO:
        createModalContentFunc = () => {
            const modalAudio = document.createElement('audio');
            modalAudio.src = timelineItem.asset_url;
            modalAudio.controls = true;
            modalAudio.preload = 'metadata';
            modalAudio.classList.add('w-70');
            return modalAudio;
        };

        break;
    // TODO(ben): Other snippet types?
    }
    
    let modalEl = null;
    itemEl.container.addEventListener('click', function() {
        if (!modalEl) {
            modalEl = modalTemplate();
            modalEl.description.innerHTML = timelineItem.description;
            modalEl.asset_container.appendChild(createModalContentFunc());

            modalEl.avatar.src = timelineItem.owner_avatar;
            modalEl.userLink.textContent = timelineItem.owner_name;
            modalEl.userLink.href = timelineItem.owner_url;
            modalEl.date.textContent = timestamp;
            modalEl.date.setAttribute("href", timelineItem.snippet_url);

            modalEl.discord_link.href = timelineItem.discord_message_url;

            function close() {
                modalEl.overlay.remove();
            }
            modalEl.overlay.addEventListener('click', close);
            modalEl.close.addEventListener('click', close);
            modalEl.container.addEventListener('click', function(e) {
                e.stopPropagation();
            });
        }

        document.body.appendChild(modalEl.overlay);
    });

    return [itemEl, doOnce(addThumbnailFunc)];
}

