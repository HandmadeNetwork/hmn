import { assert, must } from "./lib/utils";

const COUNTDOWN_INTERVAL_MS = 6000;
const COUNTDOWN_CHECK_INTERVAL_MS = 100;

const galleryContainer = must(document.querySelector<HTMLElement>(".gallery-container"));
const galleryProjects = Array.from(document.querySelectorAll<HTMLElement>(".gallery-project"));
const galleryLefts = Array.from(document.querySelectorAll<HTMLElement>(".gallery-left"));
const galleryRights = Array.from(document.querySelectorAll<HTMLElement>(".gallery-right"));
const galleryCountdown = must(document.querySelector<HTMLElement>(".gallery-countdown"));
const galleryCountdownTicks = Array.from(galleryCountdown.querySelectorAll<HTMLElement>(".gallery-countdown-tick"));

assert(galleryCountdownTicks.length > 0);

function changeGalleryProject(n: number) {
  const indexOfCurrentProject = galleryProjects.findIndex(p => !p.hidden);
  const indexOfNextProject = (indexOfCurrentProject + n + galleryProjects.length) % galleryProjects.length;
  for (let i = 0; i < galleryProjects.length; i++) {
    galleryProjects[i].hidden = i != indexOfNextProject;
  }
}

let timeRemaining = COUNTDOWN_INTERVAL_MS;
function galleryCountdownUpdate() {
  // Don't subtract any time if hovering on the entire gallery.
  if (!galleryContainer.matches(":hover")) {
    timeRemaining -= COUNTDOWN_CHECK_INTERVAL_MS;
    if (timeRemaining <= 0) {
      changeGalleryProject(1);
      timeRemaining = COUNTDOWN_INTERVAL_MS;
    }
  }

  const percentRemaining = timeRemaining / COUNTDOWN_INTERVAL_MS;
  for (const [i, tick] of galleryCountdownTicks.entries()) {
    const tickPercent = (galleryCountdownTicks.length - (i + 1)) / galleryCountdownTicks.length;
    tick.hidden = percentRemaining <= tickPercent;
  }
}
const countdownIntervalHandle = setInterval(galleryCountdownUpdate, COUNTDOWN_CHECK_INTERVAL_MS);
function stopCountdown() {
  clearInterval(countdownIntervalHandle);
  galleryCountdown.hidden = true;
}

for (const left of galleryLefts) {
  left.addEventListener("click", e => {
    e.stopPropagation();
    stopCountdown();
    changeGalleryProject(-1);
  });
}
for (const right of galleryRights) {
  right.addEventListener("click", e => {
    e.stopPropagation();
    stopCountdown();
    changeGalleryProject(1);
  });
}
