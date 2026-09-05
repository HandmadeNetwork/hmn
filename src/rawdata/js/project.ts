import { initCarousel } from "./lib/carousel";
import "./lib/relocator";

const screenshots = document.querySelector<HTMLElement>("#screenshots");
if (screenshots) {
  initCarousel(screenshots, {
    durationMS: 5000,
  });
}
