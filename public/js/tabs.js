function TabState(tabbed) {
    this.container = tabbed;
    this.tabs = tabbed.querySelector(".tab");

    this.tabbar = document.createElement("div");
    this.tabbar.classList.add("tab-bar");
    this.container.insertBefore(this.tabbar, this.container.firstChild);

    this.current_i = -1;
    this.tab_buttons = [];
}

function switch_tab_old(state, tab_i) {
    return function() {
        if (state.current_i >= 0) {
            state.tabs[state.current_i].classList.add("hidden");
            state.tab_buttons[state.current_i].classList.remove("current");
        }

        state.tabs[tab_i].classList.remove("hidden");
        state.tab_buttons[tab_i].classList.add("current");

        var hash = "";
        if (state.tabs[tab_i].hasAttribute("data-url-hash")) {
            hash = state.tabs[tab_i].getAttribute("data-url-hash");
        }
        window.location.hash = hash;

        state.current_i = tab_i;
    };
}

document.addEventListener("DOMContentLoaded", function() {
    const tabContainers = document.getElementsByClassName("tabbed");
    for (const container of tabContainers) {
        const tabBar = document.createElement("div");
        tabBar.classList.add("tab-bar");
        container.insertAdjacentElement('afterbegin', tabBar);

        const tabs = container.querySelectorAll(".tab");
        for (let i = 0; i < tabs.length; i++) {
            const tab = tabs[i];
            tab.classList.toggle('dn', i > 0);
            
            const slug = tab.getAttribute("data-slug");

            // TODO: Should this element be a link?
            const tabButton = document.createElement("div");
            tabButton.classList.add("tab-button");
            tabButton.classList.toggle("current", i === 0);
            tabButton.innerText = tab.getAttribute("data-name");
            tabButton.setAttribute("data-slug", slug);
            
            tabButton.addEventListener("click", () => {
                switchTab(container, slug);
            });

            tabBar.appendChild(tabButton);
        }

        const initialSlug = window.location.hash;
        if (initialSlug) {
            switchTab(container, initialSlug.substring(1));
        }
    }
});

function switchTab(container, slug) {
    const tabs = container.querySelectorAll('.tab');

    let didMatch = false;
    for (const tab of tabs) {
        const slugMatches = tab.getAttribute("data-slug") === slug;
        tab.classList.toggle('dn', !slugMatches);

        if (slugMatches) {
            didMatch = true;
        }
    }

    const tabButtons = document.querySelectorAll(".tab-button");
    for (const tabButton of tabButtons) {
        const buttonSlug = tabButton.getAttribute("data-slug");
        tabButton.classList.toggle('current', slug === buttonSlug);
    }

    if (!didMatch) {
        // switch to first tab as a fallback
        tabs[0].classList.remove('dn');
        tabButtons[0].classList.add('current');
    }

    window.location.hash = slug;
}

function switchToTabOfElement(container, el) {
    const tabs = Array.from(container.querySelectorAll('.tab'));
	let target = el.parentElement;
	while (target) {
		if (tabs.includes(target)) {
			switchTab(container, target.getAttribute("data-slug"));
			return;
		}
		target = target.parentElement;
	}
}
