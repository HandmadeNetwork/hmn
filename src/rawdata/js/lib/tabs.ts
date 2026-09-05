import { must } from "./utils";

export type TabsOptions = {
    /** A string identifying the tab to be automatically selected. */
    initialTab?: string,
    /** A function that is triggered when a new tab is selected. */
    onSelect?: (name: string) => boolean,
    /** Whether to fire onSelect when initializing. (Default: false) */
    fireOnSelectForInitialTab?: boolean,
};

export type TabsFunctions = {
    /** Selects a tab by name without clicking a button. */
    selectTab: (name: string, opts?: SelectTabOptions) => void,
};

export type SelectTabOptions = {
    /** If false, suppress the call to `onSelect`. Default true. */
    sendEvent?: boolean,
};

/**
 * A utility for a tabbed section on the page. To use it:
 * 
 *  - Add the `data-tab="foo"` attribute to any items you want to show and
 *    hide.
 *  - Add the `data-tab-button="foo"` attribute to any links you want to act as
 *    tab buttons. (It is expected that these links will have the `tab-button`
 *    class.)
 *  - Call this function.
 * 
 * The tab buttons will then show and hide the tab elements using the HTML
 * `hidden` attribute. The active tab button will also get the
 * `tab-button-active` class.
 */
export function initTabs(container: ParentNode, {
    initialTab,
    onSelect = () => true,
    fireOnSelectForInitialTab = false,
}: TabsOptions = {}): TabsFunctions {
    const buttons = Array.from(container.querySelectorAll("[data-tab-button]")) as HTMLElement[];
    const tabs = Array.from(container.querySelectorAll("[data-tab]")) as HTMLElement[];

    const firstTab = tabs[0].getAttribute("data-tab")!;

    function selectTab(name: string, { sendEvent = true }: SelectTabOptions = {}) {
        if (!container.querySelector(`[data-tab="${name}"]`)) {
            console.error("no tab found with name", name);
            return false;
        }

        for (const tab of tabs) {
            tab.hidden = tab.getAttribute("data-tab") !== name;
        }
        for (const button of buttons) {
            button.classList.toggle("tab-button-active", button.getAttribute("data-tab-button") === name);
        }

        if (sendEvent) {
            onSelect(name);
        }

        return true;
    }
    if (!selectTab(initialTab || firstTab, { sendEvent: fireOnSelectForInitialTab })) {
        selectTab(firstTab, { sendEvent: fireOnSelectForInitialTab });
    }

    for (const button of buttons) {
        button.addEventListener("click", () => {
            selectTab(button.getAttribute("data-tab-button")!);
        });
    }

    return {
        selectTab,
    };
}

/**
 * A wrapper around `initTabs` that automatically uses the URL #hash.
 */
export function initHashTabs(container: ParentNode, opts: TabsOptions = {}) {
    const res = initTabs(container, {
        initialTab: opts.initialTab ?? document.location.hash.substring(1),
        onSelect(name) {
            document.location.hash = `#${name}`;
            return opts.onSelect?.(name) ?? true;
        },
        fireOnSelectForInitialTab: opts.fireOnSelectForInitialTab,
    });
    const { selectTab } = res;
    window.addEventListener("hashchange", e => {
        const tab = new URL(e.newURL).hash.substring(1);
        if (tab) {
            selectTab(tab);
        }
    });

    return res;
}