const TimelineMediaTypes = {
	UNKNOWN: 0,
    IMAGE: 1,
    VIDEO: 2,
    AUDIO: 3,
    EMBED: 4,
}

const showcaseItemTemplate = makeTemplateCloner("showcase_item");
const modalTemplate = makeTemplateCloner("timeline_modal");
const tagTemplate = makeTemplateCloner("timeline_item_tag");

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

            if (timelineItem.tags.length === 0) {
                modalEl.tags.remove();
            } else {
                for (const tag of timelineItem.tags) {
                    const tagItem = tagTemplate();
                    tagItem.tag.innerText = tag.text;

                    modalEl.tags.appendChild(tagItem.root);
                }
            }

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

function initShowcaseContainer(container, items, rowHeight = 300, itemSpacing = 4) {
    const addThumbnailFuncs = new Array(items.length);

    const itemElements = []; // array of arrays
    for (let i = 0; i < items.length; i++) {
        const item = items[i];

        const [itemEl, addThumbnail] = makeShowcaseItem(item);
        itemEl.container.setAttribute('data-index', i);
        itemEl.container.setAttribute('data-date', item.date);

        addThumbnailFuncs[i] = addThumbnail;

        itemElements.push(itemEl.container);
    }

    function layout() {
        const width = container.getBoundingClientRect().width;
        container = emptyElement(container);

        function addRow(itemEls, rowWidth, container) {
            const totalSpacing = itemSpacing * (itemEls.length - 1);
            const scaleFactor = (width / Math.max(rowWidth, width));

            const row = document.createElement('div');
            row.classList.add('flex');
            row.classList.toggle('justify-between', rowWidth >= width);
            row.style.marginBottom = `${itemSpacing}px`;

            for (const itemEl of itemEls) {
                const index = parseInt(itemEl.getAttribute('data-index'), 10);
                const item = items[index];

                const aspect = item.width / item.height;
                const baseWidth = (aspect * rowHeight) * scaleFactor;
                const actualWidth = baseWidth - (totalSpacing / itemEls.length);

                itemEl.style.width = `${actualWidth}px`;
                itemEl.style.height = `${scaleFactor * rowHeight}px`;
                itemEl.style.marginRight = `${itemSpacing}px`;

                row.appendChild(itemEl);
            }

            container.appendChild(row);
        }

        let rowItemEls = [];
        let rowWidth = 0;

        for (const itemEl of itemElements) {
            const index = parseInt(itemEl.getAttribute('data-index'), 10);
            const item = items[index];

            const aspect = item.width / item.height;
            rowWidth += aspect * rowHeight;

            rowItemEls.push(itemEl);

            if (rowWidth > width) {
                addRow(rowItemEls, rowWidth, container);

                rowItemEls = [];
                rowWidth = 0;
            }
        }

        addRow(rowItemEls, rowWidth, container);
    }

    function tryLoadImages() {
        const OFFSCREEN_THRESHOLD = 0;

        const rect = container.getBoundingClientRect();
        const offscreen = (
            rect.bottom < -OFFSCREEN_THRESHOLD
            || rect.top > window.innerHeight + OFFSCREEN_THRESHOLD
        );

        if (!offscreen) {
            const items = container.querySelectorAll('.showcase-item');
            for (const item of items) {
                const i = parseInt(item.getAttribute('data-index'), 10);
                addThumbnailFuncs[i]();
            }
        }
    }

    window.addEventListener('DOMContentLoaded', () => {
        layout();
        layout(); // scrollbars are fun!!
        tryLoadImages();
    })

    window.addEventListener('resize', () => {
        layout();
        tryLoadImages();
    });

    window.addEventListener('scroll', () => {
        tryLoadImages();
    });
}
