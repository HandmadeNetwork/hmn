// src/rawdata/js/lib/utils.ts
function assert(cond, msg, soft = false) {
  if (!cond) {
    if (soft) {
      console.error(msg ?? "Assertion failed");
    } else {
      throw new Error(msg ?? "Assertion failed");
    }
  }
}

// src/rawdata/js/lib/templates.ts
var templateElementCache = {};
var templatePathCache = {};
function getTemplateEl(id) {
  if (!templateElementCache[id]) {
    const el = document.getElementById(id);
    assert(el, `no element with id ${id}`);
    assert(el instanceof HTMLTemplateElement);
    templateElementCache[id] = el;
  }
  return templateElementCache[id];
}
function getTemplatePaths(id, rootNode) {
  if (!templatePathCache[id]) {
    let descend2 = function(path, el) {
      for (var i = 0; i < el.children.length; ++i) {
        var child = el.children[i];
        var childPath = path.concat([i]);
        var tmplName = child.getAttribute("data-tmpl");
        if (tmplName) {
          paths.push([tmplName, childPath]);
        }
        if (child.children.length > 0) {
          descend2(childPath, child);
        }
      }
    };
    var descend = descend2;
    var paths = [];
    paths.push(["root", []]);
    descend2([], rootNode);
    templatePathCache[id] = paths;
  }
  return templatePathCache[id];
}
function collectElements(paths, rootElement) {
  var result = {};
  for (var i = 0; i < paths.length; ++i) {
    var path = paths[i];
    var current = rootElement;
    for (var j = 0; j < path[1].length; ++j) {
      current = current.children[path[1][j]];
    }
    result[path[0]] = current;
  }
  return result;
}
function makeTemplateCloner(id) {
  return function() {
    var templateEl = getTemplateEl(id);
    if (templateEl === null) {
      throw new Error(`Couldn't find template with ID '${id}'`);
    }
    var root = templateEl.content.cloneNode(true);
    var paths = getTemplatePaths(id, root);
    var result = collectElements(paths, root);
    return result;
  };
}
function emptyElement(el) {
  var newEl = el.cloneNode(false);
  assert(el.parentElement);
  el.parentElement.insertBefore(newEl, el);
  el.parentElement.removeChild(el);
  return newEl;
}

// src/rawdata/js/showcase.js
var TimelineMediaTypes = {
  UNKNOWN: 0,
  IMAGE: 1,
  VIDEO: 2,
  AUDIO: 3,
  EMBED: 4
};
var showcaseItemTemplate = makeTemplateCloner("showcase_item");
var modalTemplate = makeTemplateCloner("timeline_modal");
var projectLinkTemplate = makeTemplateCloner("project_link");
function showcaseTimestamp(rawDate) {
  const date = new Date(rawDate * 1e3);
  return date.toLocaleDateString([], { "dateStyle": "long" });
}
function doOnce(f) {
  let did = false;
  return () => {
    if (!did) {
      f();
      did = true;
    }
  };
}
function makeShowcaseItem(timelineItem) {
  const timestamp = showcaseTimestamp(timelineItem.date);
  const itemEl = showcaseItemTemplate();
  itemEl.avatar.style.backgroundImage = `url('${timelineItem.owner_avatar}')`;
  itemEl.username.textContent = timelineItem.owner_name;
  itemEl.when.textContent = timestamp;
  let addThumbnailFunc = () => {
  };
  let createModalContentFunc = () => {
  };
  switch (timelineItem.media_type) {
    case TimelineMediaTypes.IMAGE:
      addThumbnailFunc = () => {
        itemEl.thumbnail.style.backgroundImage = `url('${timelineItem.thumbnail_url}')`;
      };
      createModalContentFunc = () => {
        const modalImage = document.createElement("img");
        modalImage.src = timelineItem.asset_url;
        modalImage.classList.add("mw-100", "maxh-60vh");
        return modalImage;
      };
      break;
    case TimelineMediaTypes.VIDEO:
      addThumbnailFunc = () => {
        let thumbEl;
        if (timelineItem.thumbnail_url) {
          thumbEl = document.createElement("img");
          thumbEl.src = timelineItem.thumbnail_url;
        } else {
          thumbEl = document.createElement("video");
          thumbEl.src = timelineItem.asset_url;
          thumbEl.controls = false;
          thumbEl.preload = "metadata";
        }
        thumbEl.classList.add("h-100");
        itemEl.thumbnail.appendChild(thumbEl);
      };
      createModalContentFunc = () => {
        const modalVideo = document.createElement("video");
        modalVideo.src = timelineItem.asset_url;
        if (timelineItem.thumbnail_url) {
          modalVideo.poster = timelineItem.thumbnail_url;
          modalVideo.preload = "none";
        } else {
          modalVideo.preload = "metadata";
        }
        modalVideo.controls = true;
        modalVideo.classList.add("mw-100", "maxh-60vh");
        return modalVideo;
      };
      break;
    case TimelineMediaTypes.AUDIO:
      createModalContentFunc = () => {
        const modalAudio = document.createElement("audio");
        modalAudio.src = timelineItem.asset_url;
        modalAudio.controls = true;
        modalAudio.preload = "metadata";
        modalAudio.classList.add("w-70");
        return modalAudio;
      };
      break;
  }
  let modalEl = null;
  itemEl.container.addEventListener("click", function() {
    if (!modalEl) {
      let close2 = function() {
        modalEl.overlay.remove();
      };
      var close = close2;
      modalEl = modalTemplate();
      modalEl.description.innerHTML = timelineItem.description;
      modalEl.asset_container.appendChild(createModalContentFunc());
      modalEl.avatar.src = timelineItem.owner_avatar;
      modalEl.userLink.textContent = timelineItem.owner_name;
      modalEl.userLink.href = timelineItem.owner_url;
      modalEl.date.textContent = timestamp;
      modalEl.date.setAttribute("href", timelineItem.snippet_url);
      if (timelineItem.projects.length === 0) {
        modalEl.projects.remove();
      } else {
        for (const proj of timelineItem.projects) {
          const projectLink = projectLinkTemplate();
          projectLink.root.href = proj.url;
          projectLink.logo.src = proj.logo;
          projectLink.name.textContent = proj.name;
          modalEl.projects.appendChild(projectLink.root);
        }
      }
      if (timelineItem.discord_message_url != "") {
        modalEl.discord_link.href = timelineItem.discord_message_url;
      } else {
        modalEl.discord_link.remove();
      }
      modalEl.overlay.addEventListener("click", close2);
      modalEl.close.addEventListener("click", close2);
      modalEl.container.addEventListener("click", function(e) {
        e.stopPropagation();
      });
    }
    document.body.appendChild(modalEl.overlay);
  });
  return [itemEl, doOnce(addThumbnailFunc)];
}
function initShowcaseContainer(container, items, rowHeight = 300, itemSpacing = 4) {
  const addThumbnailFuncs = new Array(items.length);
  const itemElements = [];
  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    const [itemEl, addThumbnail] = makeShowcaseItem(item);
    itemEl.container.setAttribute("data-index", i);
    itemEl.container.setAttribute("data-date", item.date);
    addThumbnailFuncs[i] = addThumbnail;
    itemElements.push(itemEl.container);
  }
  function layout() {
    const width = container.getBoundingClientRect().width;
    container = emptyElement(container);
    function addRow(itemEls, rowWidth2, container2) {
      const totalSpacing = itemSpacing * (itemEls.length - 1);
      const scaleFactor = width / Math.max(rowWidth2, width);
      const row = document.createElement("div");
      row.classList.add("flex");
      row.classList.toggle("justify-between", rowWidth2 >= width);
      row.style.marginBottom = `${itemSpacing}px`;
      for (const itemEl of itemEls) {
        const index = parseInt(itemEl.getAttribute("data-index"), 10);
        const item = items[index];
        const aspect = item.width / item.height;
        const baseWidth = aspect * rowHeight * scaleFactor;
        const actualWidth = baseWidth - totalSpacing / itemEls.length;
        itemEl.style.width = `${actualWidth}px`;
        itemEl.style.height = `${scaleFactor * rowHeight}px`;
        itemEl.style.marginRight = `${itemSpacing}px`;
        row.appendChild(itemEl);
      }
      container2.appendChild(row);
    }
    let rowItemEls = [];
    let rowWidth = 0;
    for (const itemEl of itemElements) {
      const index = parseInt(itemEl.getAttribute("data-index"), 10);
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
    const offscreen = rect.bottom < -OFFSCREEN_THRESHOLD || rect.top > window.innerHeight + OFFSCREEN_THRESHOLD;
    if (!offscreen) {
      const items2 = container.querySelectorAll(".showcase-item");
      for (const item of items2) {
        const i = parseInt(item.getAttribute("data-index"), 10);
        addThumbnailFuncs[i]();
      }
    }
  }
  window.addEventListener("DOMContentLoaded", () => {
    layout();
    layout();
    tryLoadImages();
  });
  window.addEventListener("resize", () => {
    layout();
    tryLoadImages();
  });
  window.addEventListener("scroll", () => {
    tryLoadImages();
  });
}
export {
  emptyElement,
  initShowcaseContainer,
  makeShowcaseItem,
  makeTemplateCloner
};
//# sourceMappingURL=showcase.js.map
