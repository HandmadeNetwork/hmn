import { initCarousel } from "./lib/carousel";
import { assert } from "./lib/utils";

initCarousel(document.querySelector("#screenshots")!, {
  durationMS: 5000,
});

export type FollowLinkOptions = {
  csrfToken: string,
  followUrl: string,
  projectId: string,
  initialFollowing: boolean,
};

export function initFollowLink({
  csrfToken,
  followUrl,
  projectId,
  initialFollowing,
}: FollowLinkOptions) {
  const linkFollow = document.querySelector<HTMLAnchorElement>("#follow-follow");
  const linkUnfollow = document.querySelector<HTMLAnchorElement>("#follow-unfollow");
  if (!linkFollow) {
    return;
  }
  assert(linkUnfollow);

  let following = initialFollowing;
  let active = false;
  const handleFollowLink = async (e: PointerEvent) => {
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
        credentials: "include",
      });
      if (result.ok) {
        following = !following;
        linkFollow.hidden = following;
        linkUnfollow.hidden = !following;
      }
    } finally {
      active = false;
    }
  }

  linkFollow.addEventListener("click", handleFollowLink);
  linkUnfollow.addEventListener("click", handleFollowLink);
}
