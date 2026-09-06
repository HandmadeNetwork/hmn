// NOTE(ben): Validating form elements is a huge pain and TypeScript doesn't
// even have good types for it. (To be fair, the actual DOM types for this are
// asinine.)

/** Types returned by {@link HTMLFormElement.elements} */
export type FormElement =
  HTMLButtonElement
  | HTMLFieldSetElement
  | HTMLInputElement
  | HTMLObjectElement
  | HTMLOutputElement
  | HTMLSelectElement
  | HTMLTextAreaElement; // NOTE(ben): Currently no custom elements

export function firstInvalidElement(form: HTMLFormElement): FormElement | null {
  for (const el of form.elements) {
    const formEl = el as FormElement;
    if (formEl.willValidate && !formEl.validity.valid) {
      return formEl;
    }
  }
  return null;
}
