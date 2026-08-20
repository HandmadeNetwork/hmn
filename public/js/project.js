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
    const items = document.querySelectorAll(".carousel-item");
    for (const item of items) {
      item.classList.remove("active");
    }
    items[i].classList.add("active");
    const smallItems = document.querySelectorAll(".carousel-item-small");
    if (smallItems.length > 0) {
      for (const item of smallItems) {
        item.classList.remove("active");
      }
      smallItems[i].classList.add("active");
    }
    const buttons = document.querySelectorAll(".carousel-button");
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

// src/rawdata/js/project.ts
initCarousel(document.querySelector("#screenshots"), {
  durationMS: 5e3
});
function initFollowLink({
  csrfToken,
  followUrl,
  projectId,
  initialFollowing
}) {
  const linkFollow = document.querySelector("#follow-follow");
  const linkUnfollow = document.querySelector("#follow-unfollow");
  if (!linkFollow) {
    return;
  }
  assert(linkUnfollow);
  let following = initialFollowing;
  let active = false;
  const handleFollowLink = async (e) => {
    e.preventDefault();
    if (active) {
      return;
    }
    try {
      active = true;
      let formData = new FormData();
      formData.set("csrf_token", csrfToken);
      formData.set("project_id", projectId);
      if (following) {
        formData.set("unfollow", "true");
      }
      let result = await fetch(followUrl, {
        method: "POST",
        body: formData,
        redirect: "error",
        credentials: "include"
      });
      if (result.ok) {
        following = !following;
        linkFollow.hidden = following;
        linkUnfollow.hidden = !following;
      }
    } finally {
      active = false;
    }
  };
  linkFollow.addEventListener("click", handleFollowLink);
  linkUnfollow.addEventListener("click", handleFollowLink);
}
export {
  initFollowLink
};
//# sourceMappingURL=project.js.map
