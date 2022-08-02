function initCarousel(container, options = {}) {
    const durationMS = options.durationMS ?? 0;
    const onChange = options.onChange ?? (() => {});

    const numCarouselItems = container.querySelectorAll('.carousel-item').length;
    const buttonContainer = container.querySelector('.carousel-buttons');

    let current = 0;
    function activateCarousel(i) {
        const items = document.querySelectorAll('.carousel-item');
        for (const item of items) {
            item.classList.remove('active');
        }
        items[i].classList.add('active');

        const smallItems = document.querySelectorAll('.carousel-item-small');
        if (smallItems.length > 0) {
            for (const item of smallItems) {
                item.classList.remove('active');
            }
            smallItems[i].classList.add('active');
        }

        const buttons = document.querySelectorAll('.carousel-button');
        for (const button of buttons) {
            button.classList.remove('active');
        }
        buttons[i].classList.add('active');

        current = i;

        onChange(current);
    }

    function activateNext() {
        activateCarousel((current + numCarouselItems + 1) % numCarouselItems);
    }

    function activatePrev() {
        activateCarousel((current + numCarouselItems - 1) % numCarouselItems);
    }

    const carouselTimer = durationMS > 0 && setInterval(() => {
        if (numCarouselItems === 0) {
            return;
        }
        activateNext();
    }, durationMS);

    function carouselButtonClick(i) {
        activateCarousel(i);
        if (carouselTimer) {
            clearInterval(carouselTimer);
        }
    }

    for (let i = 0; i < numCarouselItems; i++) {
        const button = document.createElement('div');
        button.classList.add('carousel-button', 'br-pill', 'w1', 'h1', 'mh2');
        button.classList.toggle('active', i === 0);

        const clickIndex = i;
        button.addEventListener('click', () => {
            carouselButtonClick(clickIndex);
        });

        buttonContainer.appendChild(button);
    }

    activateCarousel(0);

    return {
        next: activateNext,
        prev: activatePrev,
    };
}
