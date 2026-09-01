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

// src/rawdata/js/project_index.ts
var COUNTDOWN_INTERVAL_MS = 6e3;
var COUNTDOWN_CHECK_INTERVAL_MS = 100;
var galleryContainer = must(document.querySelector(".gallery-container"));
var galleryProjects = Array.from(document.querySelectorAll(".gallery-project"));
var galleryLefts = Array.from(document.querySelectorAll(".gallery-left"));
var galleryRights = Array.from(document.querySelectorAll(".gallery-right"));
var galleryCountdown = must(document.querySelector(".gallery-countdown"));
var galleryCountdownTicks = Array.from(galleryCountdown.querySelectorAll(".gallery-countdown-tick"));
assert(galleryCountdownTicks.length > 0);
function changeGalleryProject(n) {
  const indexOfCurrentProject = galleryProjects.findIndex((p) => !p.hidden);
  const indexOfNextProject = (indexOfCurrentProject + n + galleryProjects.length) % galleryProjects.length;
  for (let i = 0; i < galleryProjects.length; i++) {
    galleryProjects[i].hidden = i != indexOfNextProject;
  }
}
var timeRemaining = COUNTDOWN_INTERVAL_MS;
function galleryCountdownUpdate() {
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
var countdownIntervalHandle = setInterval(galleryCountdownUpdate, COUNTDOWN_CHECK_INTERVAL_MS);
function stopCountdown() {
  clearInterval(countdownIntervalHandle);
  galleryCountdown.hidden = true;
}
for (const left of galleryLefts) {
  left.addEventListener("click", (e) => {
    e.stopPropagation();
    stopCountdown();
    changeGalleryProject(-1);
  });
}
for (const right of galleryRights) {
  right.addEventListener("click", (e) => {
    e.stopPropagation();
    stopCountdown();
    changeGalleryProject(1);
  });
}
//# sourceMappingURL=project_index.js.map
