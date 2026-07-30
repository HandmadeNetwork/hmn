import { autosaveContent, initLiveMarkdown } from "./lib/markdown_previews";
import { setupMarkdownUpload } from "./lib/markdown_upload";
import { HTMLFileInputElement } from "./lib/types";
import { must } from "./lib/utils";

const form = must(document.querySelector<HTMLFormElement>('#form'));
const titleField = document.querySelector<HTMLInputElement>('#title'); // may be null in cases like replies
const textField = must(document.querySelector<HTMLTextAreaElement>('#editor'));
const preview = must(document.querySelector<HTMLElement>('#preview'));

export type EditorOptions = {
  maxFileSize: number,
  uploadUrl: string,
  markdownParser?: string,
};

export function init({ maxFileSize, uploadUrl, markdownParser }: EditorOptions) {
  // Save content on change, clear on submit
  const clearFuncs: (() => void)[] = [];
  if (titleField) {
    const { clear: clearTitle } = autosaveContent({
      inputEl: titleField,
      storageKey: `post-title/${window.location.host}${window.location.pathname}`,
    });
    clearFuncs.push(clearTitle);
  }
  const { clear: clearContent } = autosaveContent({
    inputEl: textField,
    storageKey: `post-content/${window.location.host}${window.location.pathname}`,
  });
  clearFuncs.push(clearContent);
  form.addEventListener('submit', e => {
    for (const clear of clearFuncs) {
      clear();
    }
  });

  // Do live Markdown previews
  const doMarkdown = initLiveMarkdown({
    inputEl: textField,
    previewEl: preview,
    parserName: markdownParser,
  });

  /*
  / Asset upload
  */
  setupMarkdownUpload(
    document.querySelectorAll("#form input[type=submit]"),
    must(document.querySelector<HTMLFileInputElement>('#file_input')),
    must(document.querySelector('.upload_bar')),
    textField,
    doMarkdown,
    maxFileSize,
    uploadUrl
  );

  textField.addEventListener("keydown", function (ev) {
    if (ev.ctrlKey) {
      const key = ev.key.toLowerCase();
      const start = textField.selectionStart;
      const end = textField.selectionEnd;
      const selected = textField.value.substring(start, end);

      if (key === 'k') {
        ev.preventDefault();

        if (selected.length) {
          textField.value =
            textField.value.substring(0, start) + '[' +
            selected + ']()' +
            textField.value.substring(end);

          textField.selectionStart = textField.selectionEnd = end + 3;
        }
        else {
          textField.value =
            textField.value.substring(0, start) +
            "[](url)" +
            textField.value.substring(end);

          textField.selectionStart = textField.selectionEnd = start + 1;
        }
        textField.focus();
        doMarkdown();
      }
      else if (key === 'b') {
        ev.preventDefault();

        if (selected.length) {
          textField.value =
            textField.value.substring(0, start) + '**' +
            selected + '**' +
            textField.value.substring(end);

          textField.selectionStart = textField.selectionEnd = end + 2;
        }
        else {
          textField.value =
            textField.value.substring(0, start) +
            "****" +
            textField.value.substring(end);

          textField.selectionStart = textField.selectionEnd = start + 2;
        }
        textField.focus();
        doMarkdown();
      }
      else if (key === 'i') {
        ev.preventDefault();

        if (selected.length) {
          textField.value =
            textField.value.substring(0, start) + '*' +
            selected + '*' +
            textField.value.substring(end);

          textField.selectionStart = textField.selectionEnd = end + 1;
        }
        else {
          textField.value =
            textField.value.substring(0, start) +
            "**" +
            textField.value.substring(end);

          textField.selectionStart = textField.selectionEnd = start + 1;
        }
        textField.focus();
        doMarkdown();
      }
      else if (key === 's') {
        ev.preventDefault();
      }
    }
  });

  textField.addEventListener("paste", (ev) => {
    const clipboard = must(ev.clipboardData);
    const pastedText = clipboard.getData("text");
    const start = textField.selectionStart;
    const end = textField.selectionEnd;

    if (pastedText && (pastedText.startsWith('https://') || pastedText.startsWith('http://')) && start !== end) {
      ev.preventDefault();

      textField.value =
        textField.value.substring(0, start) + '[' +
        textField.value.substring(start, end) + '](' +
        pastedText + ')' +
        textField.value.substring(end);

      textField.selectionStart = textField.selectionEnd = end + 4 + pastedText.length;

      textField.focus();
      doMarkdown();
    }
  });
}
