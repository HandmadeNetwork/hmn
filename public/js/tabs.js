function initTabs(container, {
    initialTab = null,
    onSelect = (name) => {},
}) {
    const buttons = Array.from(container.querySelectorAll("[data-tab-button]"));
    const tabs = Array.from(container.querySelectorAll("[data-tab]"));

    const firstTab = tabs[0].getAttribute("data-tab");

    function selectTab(name, { sendEvent = true } = {}) {
        if (!document.querySelector(`[data-tab="${name}"]`)) {
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
