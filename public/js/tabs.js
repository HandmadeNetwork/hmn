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
 * 
 * You can provide the following options in the second argument:
 * 
 *  - `initialTab`: A string identifying a tab to be automatically selected.
 *  - `onSelect`: A function of type `(name: string) => void` that is triggered
 *    when a new tab is selected.
 * 
 * Returns an object with the following:
 * 
 *  - `selectTab(name, options?: { sendEvent?: bool = true })`:
 *    A function that selects a tab by name without clicking a button. By
 *    setting `sendEvent` to false you can suppress the call to `onSelect`.
 */
function initTabs(container, {
    initialTab = null,
    onSelect = name => {},
} = {}) {
    const buttons = Array.from(container.querySelectorAll("[data-tab-button]"));
    const tabs = Array.from(container.querySelectorAll("[data-tab]"));

    const firstTab = tabs[0].getAttribute("data-tab");

    function selectTab(name, { sendEvent = true } = {}) {
        if (!container.querySelector(`[data-tab="${name}"]`)) {
            console.warn("no tab found with name", name);
            return selectTab(firstTab, initial);
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
    }
    selectTab(initialTab || firstTab, { sendEvent: false });

    for (const button of buttons) {
        button.addEventListener("click", () => {
            selectTab(button.getAttribute("data-tab-button"));
        });
    }

    return {
        selectTab,
    };
}

/**
 * A wrapper around the above that automatically uses the URL #hash.
 */
function initHashTabs(container, {
    initialTab = null,
} = {}) {
    const res = initTabs(container, {
        initialTab: initialTab ?? document.location.hash.substring(1),
        onSelect(name) {
            document.location.hash = `#${name}`;
        },
    });
    const { selectTab } = res;
    window.addEventListener("hashchange", e => {
        const tab = new URL(e.newURL).hash.substring(1);
        if (tab) {
            selectTab(tab, { sendEvent: false });
        }
    });
    
    return res;
}