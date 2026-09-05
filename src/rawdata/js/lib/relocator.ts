import { LARGE_EM, MEDIUM_EM } from "./constants";
import { must } from "./utils";

export function updateRelocators() {
  const l = window.matchMedia(`(min-width: ${LARGE_EM}em)`).matches;
  const ns = window.matchMedia(`(min-width: ${MEDIUM_EM}em)`).matches;
  for (const relocator of document.querySelectorAll<HTMLElement>(".relocator")) {
    const targetDefaultID = must(relocator.dataset.relocate, "must have a data-relocate attribute");
    const targetNSID = relocator.dataset.relocateNs ?? targetDefaultID;
    const targetLID = relocator.dataset.relocateL ?? targetNSID;

    const targetDefault = must(document.getElementById(targetDefaultID), `no element found with id ${targetDefaultID}`);
    const targetNS = must(document.getElementById(targetNSID), `no element found with id ${targetNSID}`);
    const targetL = must(document.getElementById(targetLID), `no element found with id ${targetLID}`);

    const target = l ? targetL : (ns ? targetNS : targetDefault);

    if (relocator.previousSibling !== target) {
      // NOTE(ben): My kingdom for an `insertAfter`.
      target.parentElement!.insertBefore(relocator, target.nextSibling);
    }
  }
}
updateRelocators();
window.addEventListener("resize", updateRelocators);
