/**
 * Sets up a UI where items can be dragged and dropped vertically to reorder
 * them. Use it like so:
 *
 *  1. Give all the relevant items the `reorderable-item` class.
 *  2. Make sure that the items' parent element is positioned, because the
 *     dragged item will use `position: absolute`. (Usually `relative` is
 *     fine.)
 *  3. Call `initReorderable` and get the `startDrag` function it returns.
 *  4. Add `startDrag` as a `pointerdown` event listener on each item. This is
 *     not automatic because you may wish to apply it only to e.g. a drag
 *     handle, or not at all (if for example an item needs to move around but
 *     cannot be dragged explicitly).
 *
 * The `onReorder` event will be called while you drag whenever items change
 * order. This can be used to update previews or whatever else. During an
 * `onReorder`, all items will be present in the DOM as if the drag had
 * finished; however, there will also be an item with the `reorderable-dummy`
 * class that you may need to be aware of.
 */
export function initReorderable(container, {
  onReorder = item => { },
} = {}) {
  let dragItem = null;
  let dragPointerId = null;
  let dragItemStartY = 0;
  let dragMouseStartY = 0;

  const dummy = document.createElement("div");
  dummy.classList.add("reorderable-dummy");

  function startDrag(e) {
    if (!e.isPrimary || e.button !== 0) {
      return;
    }
    e.preventDefault();

    const item = e.target.closest(".reorderable-item");

    const top = item.offsetTop;
    item.style.position = "absolute";
    item.style.top = `${top}px`;
    item.classList.add("reorderable-dragging");

    dummy.style.height = `${item.offsetHeight}px`;
    item.insertAdjacentElement("beforebegin", dummy);
    document.body.classList.add("grabbing");

    dragItem = item;
    dragPointerId = e.pointerId;
    dragItemStartY = top;
    dragMouseStartY = e.pageY;

    container.setPointerCapture(e.pointerId);
    container.addEventListener("pointermove", doDrag);
    container.addEventListener("lostpointercapture", endDrag, { once: true });
  }

  function doDrag(e) {
    const delta = e.pageY - dragMouseStartY;
    const top = dragItemStartY + delta;
    const middle = top + dragItem.offsetHeight / 2;

    // NOTE(ben): Find the closest item to insert before or after.
    const items = container.querySelectorAll(".reorderable-item");
    let closestItem = null;
    let closestItemDist = Infinity;
    let insertBefore = null; // true means before, false means after
    for (const item of items) {
      if (item === dragItem) {
        continue;
      }

      const itemMiddle = item.offsetTop + item.offsetHeight / 2;
      const dist = middle - itemMiddle;
      if (Math.abs(dist) < closestItemDist) {
        closestItem = item;
        closestItemDist = Math.abs(dist);
        insertBefore = dist < 0;
      }
    }

    if (closestItem) {
      // NOTE(ben): Suppress reordering if possible. Walk forward or backward
      // looking for the drag item; if we find it first, no reordering.
      let alreadyOrdered = true;
      let n = closestItem;
      while (true) {
        if (insertBefore) {
          n = n.previousSibling;
        } else {
          n = n.nextSibling;
        }

        if (!n) {
          alreadyOrdered = false;
          break;
        }
        if (n === dragItem) {
          break;
        }
        if (n.classList?.contains("reorderable-item")) {
          alreadyOrdered = false;
          break;
        }
      }

      if (!alreadyOrdered) {
        closestItem.insertAdjacentElement(insertBefore ? "beforebegin" : "afterend", dummy);
        dragItem.remove();
        dummy.insertAdjacentElement("beforebegin", dragItem);
        onReorder(dragItem);
      }
    }

    // NOTE(ben): Do this at the end so the final position is accurately
    // clamped. (I don't really like that it comes after calling onReorder; I'd
    // prefer if the entire thing was done when we fired the event. But the
    // problem is that we use onReorder to actually update the presence of
    // placeholders, etc., so we really do need to do that before doing this
    // clamp, or else we can incur one frame of bad clampage.)
    const maxTop = container.offsetHeight - dragItem.offsetHeight;
    const newTop = Math.max(0, Math.min(maxTop, top));
    dragItem.style.top = `${newTop}px`;
  }

  function endDrag(e) {
    container.removeEventListener("pointermove", doDrag);

    dragItem.remove();
    dummy.insertAdjacentElement("beforebegin", dragItem);
    dummy.remove();

    dragItem.style.position = null;
    dragItem.style.top = null;
    dragItem.classList.remove("reorderable-dragging");

    document.body.classList.remove("grabbing");

    onReorder(dragItem);

    dragItem = null;
    dragPointerId = null;
    dragItemStartY = 0;
    dragMouseStartY = 0;
  }

  return {
    startDrag,
  };
}
