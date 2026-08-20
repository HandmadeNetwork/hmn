// src/rawdata/js/lib/tabs.ts
function initTabs(container, {
  initialTab,
  onSelect = () => {
  }
} = {}) {
  const buttons = Array.from(container.querySelectorAll("[data-tab-button]"));
  const tabs = Array.from(container.querySelectorAll("[data-tab]"));
  const firstTab = tabs[0].getAttribute("data-tab");
  function selectTab(name, { sendEvent = true } = {}) {
    if (!container.querySelector(`[data-tab="${name}"]`)) {
      console.warn("no tab found with name", name);
      return selectTab(firstTab, { sendEvent });
    }
    for (const tab of tabs) {
      tab.hidden = tab.getAttribute("data-tab") !== name;
    }
    for (const button of buttons) {
      button.classList.toggle("tab-button-active", button.getAttribute("data-tab-button") === name);
    }
    if (sendEvent) {
      onSelect(name);
    }
  }
  selectTab(initialTab || firstTab, { sendEvent: false });
  for (const button of buttons) {
    button.addEventListener("click", () => {
      selectTab(button.getAttribute("data-tab-button"));
    });
  }
  return {
    selectTab
  };
}
function initHashTabs(container, {
  initialTab
} = {}) {
  const res = initTabs(container, {
    initialTab: initialTab ?? document.location.hash.substring(1),
    onSelect(name) {
      document.location.hash = `#${name}`;
    }
  });
  const { selectTab } = res;
  window.addEventListener("hashchange", (e) => {
    const tab = new URL(e.newURL).hash.substring(1);
    if (tab) {
      selectTab(tab, { sendEvent: false });
    }
  });
  return res;
}

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
function must(val, msg) {
  assert(val, msg);
  return val;
}

// src/rawdata/js/landing.ts
var noFollowing = document.querySelectorAll("[data-tab='following'] .timeline-item").length === 0;
initHashTabs(must(document.querySelector("#landing-tabs")), {
  initialTab: document.location.hash.substring(1) || (noFollowing ? "featured" : void 0)
});
var latestNews = must(document.querySelector("#latest_news"));
var latestNewsPostID = must(latestNews.getAttribute("data-id"));
var latestNewsClosedKey = "latest_news_closed";
document.querySelector("#close-latest-news-button").addEventListener("click", (e) => {
  e.preventDefault();
  localStorage.setItem(latestNewsClosedKey, latestNewsPostID);
  hideLatestNewsIfClosedOrRead();
});
function hideLatestNewsIfClosedOrRead() {
  const isUnread = latestNews.hasAttribute("data-unread");
  const closedID = localStorage.getItem(latestNewsClosedKey);
  if (!isUnread || closedID === latestNewsPostID) {
    latestNews.hidden = true;
  }
}
hideLatestNewsIfClosedOrRead();
//# sourceMappingURL=landing.js.map
