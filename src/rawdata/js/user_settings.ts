import { ImageSelector } from "./lib/image_selector";
import { initLinkEditor } from "./lib/link_editor";
import { Asset } from "./lib/models";
import { initHashTabs } from "./lib/tabs";
import { must } from "./lib/utils";

function lengthReporter(inputEl: HTMLInputElement, lengthEl: HTMLElement) {
  let updateLength = function () {
    lengthEl.textContent = `${inputEl.value.length}/${inputEl.getAttribute("maxlength")}`;
  }
  inputEl.addEventListener("input", updateLength);
  updateLength();
}

export type UserSettingsOptions = {
  avatarMaxFileSize: number,
  avatar: Asset | null,
  linksJson: string,
};

export function init({
  avatarMaxFileSize,
  avatar,
  linksJson,
}: UserSettingsOptions) {
  lengthReporter(
    must(document.getElementById("realname")) as HTMLInputElement,
    must(document.querySelector(".realname-length")),
  );

  let avatarSelector = new ImageSelector(
    "user_avatar",
    avatarMaxFileSize,
    {
      original: avatar || undefined,
    },
  );

  must(document.querySelector("#upload-avatar-button")).addEventListener("click", e => {
    e.preventDefault();
    avatarSelector.openImageInput();
  });
  must(document.querySelector("#user-avatar-placeholder")).replaceWith(avatarSelector.root);

  initLinkEditor(JSON.parse(linksJson));

  lengthReporter(
    must(document.getElementById("shortbio")) as HTMLInputElement,
    must(document.querySelector(".shortbio-length")),
  );

  lengthReporter(
    must(document.getElementById("longbio")) as HTMLInputElement,
    must(document.querySelector(".longbio-length")),
  );

  if (document.getElementById("signature")) {
    lengthReporter(
      must(document.getElementById("signature")) as HTMLInputElement,
      must(document.querySelector(".signature-length")),
    );
  }

  initHashTabs(document);

  const discordUnlinkForm = must(document.querySelector<HTMLFormElement>('#discord-unlink-form'));
  document.querySelector("#unlink-discord-button")?.addEventListener("click", e => {
    e.preventDefault();
    discordUnlinkForm.submit();
  });

  const discordShowcaseBacklogForm = must(document.querySelector<HTMLFormElement>("#discord-showcase-backlog"));
  document.querySelector("#discord-showcase-backlog-button")?.addEventListener("click", e => {
    e.preventDefault();
    discordShowcaseBacklogForm.submit();
  });
}
