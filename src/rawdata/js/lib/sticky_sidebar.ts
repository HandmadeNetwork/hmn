/*
 * This script can be used to lay out a sidebar alongside the main content
 * where sidebar elements are positioned and sized vertically to align with
 * named elements in the main content. That was a useless mouthful so here is a
 * diagram:
 *
 *            Main  │  Sidebar
 *  ┌────────────┐  │  ┌────────┐
 *  │a           │  │  │top=a   │
 *  └────────────┘  │  │        │
 *  ┌────────────┐  │  │        │
 *  │b           │  │  │        │
 *  │            │  │  │        │
 *  │            │  │  │bottom=b│
 *  └────────────┘  │  └────────┘
 *  ┌────────────┐  │  ┌────────┐
 *  │c           │  │  │top=c   │
 *  └────────────┘  │  │        │
 *  ┌─────┐┌─────┐  │  │        │
 *  │d    ││e    │  │  │bottom=e│
 *  └─────┘└─────┘  │  └────────┘
 *
 * The main content is laid out like normal. The sidebar items are absolutely
 * positioned and configured with top and bottom IDs. This script will keep the
 * sidebar items positioned and sized so that the top and bottom line up with
 * the named elements.
 *
 * This kind of layout is ideal for sticky positioning, since you can easily
 * craft a box to contain the sticky element so that it comes on screen and
 * scrolls off screen at the right time. Or you can just use this to absolutely
 * position elements next to each other.
 *
 * To use this:
 * 
 *    1. Import this script.
 *    2. Add the "stickinator" class to the sidebar elements you want
 *       positioned. Also add data-top and data-bottom attributes with the IDs
 *       of the main content elements to align to. (If you omit either, it will
 *       default to the top or bottom or the sidebar respectively.)
 *    3. Make sure the sidebar element has `position: relative`, so that
 *       elements can be absolutely positioned inside.
 *    4. Make sure the `.stickinator` elements are all absolutely positioned.
 *       (This script will not do that for you.)
 *
 * No further action is strictly necessary; `.stickinator` elements will have
 * their `top` and `height` will be set automatically on window resize.
 */

/**
 * Updates all sticky sidebars. Can be called at any time if you know an update
 * is required.
 */
export function updateStickySidebars() {
  for (const container of document.querySelectorAll<HTMLElement>(".stickinator")) {
    const parent = container.parentElement!;
    const parentTopScreen = parent.getBoundingClientRect().top;
    const parentHeight = parent.getBoundingClientRect().height;

    const topID = container.dataset.top;
    const bottomID = container.dataset.bottom;
    const topEl = topID ? document.getElementById(topID) : null;
    const bottomEl = bottomID ? document.getElementById(bottomID) : null;

    let topInParent, height;
    if (topEl) {
      const topTopScreen = topEl.getBoundingClientRect().top;
      topInParent = topTopScreen - parentTopScreen;
    } else {
      topInParent = 0;
    }
    if (bottomEl) {
      const bottomBottomScreen = bottomEl.getBoundingClientRect().bottom;
      height = bottomBottomScreen - parentTopScreen - topInParent;
    } else {
      height = parentHeight - topInParent;
    }

    container.style.top = `${topInParent}px`;
    container.style.height = `${height}px`;
  }
}
window.addEventListener("resize", updateStickySidebars);
