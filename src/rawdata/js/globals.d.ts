declare const MathJax: {
  // NOTE(ben): These are declared as optional parameters because the MathJax
  // global is defined immediately but not filled out until mathjax.js finishes
  // loading (asynchronously).
  typeset?: (elements?: HTMLElement[]) => void;
  typesetPromise?: (elements?: HTMLElement[]) => Promise<void>;
};
