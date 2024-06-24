function initTabs(container, initialTab = null) {
    const buttons = Array.from(container.querySelectorAll("[data-tab-button]"));
    const tabs = Array.from(container.querySelectorAll("[data-tab]"));

    if (!initialTab) {
        initialTab = tabs[0].getAttribute("data-tab");
    }

    function switchTo(name) {
        for (const tab of tabs) {
            tab.hidden = tab.getAttribute("data-tab") !== name;
        }
        for (const button of buttons) {
            button.classList.toggle("tab-button-active", button.getAttribute("data-tab-button") === name);
        }
    }
    switchTo(initialTab);

    for (const button of buttons) {
        button.addEventListener("click", () => {
            switchTo(button.getAttribute("data-tab-button"));
        });
    }
}
