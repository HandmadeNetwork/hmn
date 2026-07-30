import { initHashTabs } from "./lib/tabs";
import { must } from "./lib/utils";

const noFollowing = document.querySelectorAll("[data-tab='following'] .timeline-item").length === 0;
initHashTabs(must(document.querySelector("#landing-tabs")), {
  initialTab: document.location.hash.substring(1) || (noFollowing ? "featured" : undefined),
});

// Latest news

const latestNews = must(document.querySelector<HTMLElement>("#latest_news"));
const latestNewsPostID = must(latestNews.getAttribute("data-id"));
const latestNewsClosedKey = "latest_news_closed";

function closeLatestNews(e) {
  e.preventDefault();
  localStorage.setItem(latestNewsClosedKey, latestNewsPostID);
  hideLatestNewsIfClosedOrRead();
}

function hideLatestNewsIfClosedOrRead() {
  const isUnread = latestNews.hasAttribute("data-unread");
  const closedID = localStorage.getItem(latestNewsClosedKey);
  if (!isUnread || closedID === latestNewsPostID) {
    latestNews.hidden = true;
  }
}
hideLatestNewsIfClosedOrRead();
