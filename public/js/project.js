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

// src/rawdata/js/lib/carousel.ts
function initCarousel(container, options = {}) {
  const durationMS = options.durationMS ?? 0;
  const onChange = options.onChange ?? (() => {
  });
  const numCarouselItems = container.querySelectorAll(".carousel-item").length;
  const buttonContainer = must(container.querySelector(".carousel-buttons"));
  let current = 0;
  function activateCarousel(i, silent = false) {
    const items = container.querySelectorAll(".carousel-item");
    for (const item of items) {
      item.classList.remove("active");
    }
    items[i].classList.add("active");
    const smallItems = container.querySelectorAll(".carousel-item-small");
    if (smallItems.length > 0) {
      for (const item of smallItems) {
        item.classList.remove("active");
      }
      smallItems[i].classList.add("active");
    }
    const buttons = container.querySelectorAll(".carousel-button");
    for (const button of buttons) {
      button.classList.remove("active");
    }
    buttons[i].classList.add("active");
    current = i;
    if (!silent) {
      onChange(current);
    }
  }
  function activateNext() {
    activateCarousel((current + numCarouselItems + 1) % numCarouselItems);
  }
  function activatePrev() {
    activateCarousel((current + numCarouselItems - 1) % numCarouselItems);
  }
  const carouselTimer = durationMS > 0 && setInterval(() => {
    if (numCarouselItems === 0) {
      return;
    }
    activateNext();
  }, durationMS);
  function carouselButtonClick(i) {
    activateCarousel(i);
    if (carouselTimer) {
      clearInterval(carouselTimer);
    }
  }
  for (let i = 0; i < numCarouselItems; i++) {
    const button = document.createElement("div");
    button.classList.add("carousel-button");
    button.classList.toggle("active", i === 0);
    const clickIndex = i;
    button.addEventListener("click", () => {
      carouselButtonClick(clickIndex);
    });
    buttonContainer.appendChild(button);
  }
  activateCarousel(0, true);
  return {
    next: activateNext,
    prev: activatePrev
  };
}

// src/rawdata/js/lib/constants.ts
var MEDIUM_EM = 35;
var LARGE_EM = 60;

// src/rawdata/js/lib/relocator.ts
function updateRelocators() {
  const l = window.matchMedia(`(min-width: ${LARGE_EM}em)`).matches;
  const ns = window.matchMedia(`(min-width: ${MEDIUM_EM}em)`).matches;
  for (const relocator of document.querySelectorAll(".relocator")) {
    const targetDefaultID = must(relocator.dataset.relocate, "must have a data-relocate attribute");
    const targetNSID = relocator.dataset.relocateNs ?? targetDefaultID;
    const targetLID = relocator.dataset.relocateL ?? targetNSID;
    const targetDefault = must(document.getElementById(targetDefaultID), `no element found with id ${targetDefaultID}`);
    const targetNS = must(document.getElementById(targetNSID), `no element found with id ${targetNSID}`);
    const targetL = must(document.getElementById(targetLID), `no element found with id ${targetLID}`);
    const target = l ? targetL : ns ? targetNS : targetDefault;
    if (relocator.previousSibling !== target) {
      target.parentElement.insertBefore(relocator, target.nextSibling);
    }
  }
}
updateRelocators();
window.addEventListener("resize", updateRelocators);

// src/rawdata/js/project.ts
var screenshots = document.querySelector("#screenshots");
if (screenshots) {
  initCarousel(screenshots, {
    durationMS: 5e3
  });
}
//# sourceMappingURL=project.js.map
