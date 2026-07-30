import { ImageSelector } from "./lib/image_selector";
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
  avatarUrl: string | undefined,
  avatarFilename: string | undefined,
};

export function init({
  avatarMaxFileSize,
  avatarUrl,
  avatarFilename,
}: UserSettingsOptions) {
  lengthReporter(
    must(document.getElementById("realname")) as HTMLInputElement,
    must(document.querySelector(".realname-length")),
  );

  let avatarSelector = new ImageSelector(
    "user_avatar",
    avatarMaxFileSize,
    {
      originalUrl: avatarUrl,
      originalFilename: avatarFilename,
    },
  );

  must(document.querySelector("#upload-avatar-button")).addEventListener("click", e => {
    e.preventDefault();
    avatarSelector.openImageInput();
  });
  must(document.querySelector("#user-avatar-placeholder")).replaceWith(avatarSelector.root);

  lengthReporter(
    must(document.getElementById("shortbio")) as HTMLInputElement,
    must(document.querySelector(".shortbio-length")),
  );

  lengthReporter(
    must(document.getElementById("longbio")) as HTMLInputElement,
    must(document.querySelector(".longbio-length")),
  );

  lengthReporter(
    must(document.getElementById("signature")) as HTMLInputElement,
    must(document.querySelector(".signature-length")),
  );

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
