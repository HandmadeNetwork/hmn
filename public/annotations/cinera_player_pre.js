// refsCallback: (optional)
//   Will be called when the player enters a marker that has a `data-ref` attribute. The value of `data-ref` will be passed to the function.
//   When leaving a marker that a `data-ref` attribute, and entering a marker without one (or not entering a new marker at all), the function will be called with `null`.
function Player(htmlContainer, refsCallback) {
    this.container = htmlContainer;
    this.markersContainer = this.container.querySelector(".markers_container");
    this.videoContainer = this.container.querySelector(".video_container");
    this.refsCallback = refsCallback || function() {};

    if (!this.videoContainer.getAttribute("data-videoId")) {
        console.error("Expected to find data-videoId attribute on", this.videoContainer, "for player initialized on", this.container);
        throw new Error("Missing data-videoId attribute.");
    }
    this.markers = [];
    var markerEls = this.markersContainer.querySelectorAll(".marker");
    if (markerEls.length == 0) {
        console.error("No markers found in", this.markersContainer, "for player initialized on", this.container);
        throw new Error("Missing markers.");
    }
    for (var i = 0; i < markerEls.length; ++i) {
        var marker = {
            timestamp: parseInt(markerEls[i].getAttribute("data-timestamp"), 10),
            ref: markerEls[i].getAttribute("data-ref"),
            endTime: (i < markerEls.length - 1 ? parseInt(markerEls[i+1].getAttribute("data-timestamp"), 10) : null),
            el: markerEls[i],
            fadedProgress: markerEls[i].querySelector(".progress.faded"),
            progress: markerEls[i].querySelector(".progress.main"),
            hoverx: null
        };
        marker.el.addEventListener("click", this.onMarkerClick.bind(this, marker));
        marker.el.addEventListener("mousemove", this.onMarkerMouseMove.bind(this, marker));
        marker.el.addEventListener("mouseleave", this.onMarkerMouseLeave.bind(this, marker));
        this.markers.push(marker);
    }

    this.currentMarker = null;
    this.currentMarkerIdx = null;
    this.youtubePlayer = null;
    this.youtubePlayerReady = false;
    this.playing = false;
    this.shouldPlay = false;
    this.buffering = false;
    this.pauseAfterBuffer = false;
    this.speed = 1;
    this.currentTime = -1;
    this.scrollTo = -1;
    this.scrollPosition = 0;
    this.nextFrame = null;
    this.looping = false;


    this.markersContainer.addEventListener("wheel", function(ev) {
        this.scrollTo = -1;
    }.bind(this));

    Player.initializeYoutube(this.onYoutubeReady.bind(this));
    this.updateSize();
    this.resume();
}

// Start playing the video from the current position.
// If the player hasn't loaded yet, it will autoplay when ready.
Player.prototype.play = function() {
    if (this.youtubePlayerReady) {
        if (!this.playing) {
            this.youtubePlayer.playVideo();
        }
        this.pauseAfterBuffer = false;
    } else {
        this.shouldPlay = true;
    }
};

// Pause the video at the current position.
// If the player hasn't loaded yet, it will not autoplay when ready. (This is the default)
Player.prototype.pause = function() {
    if (this.youtubePlayerReady) {
        if (this.playing) {
            this.youtubePlayer.pauseVideo();
        } else if (this.buffering) {
            this.pauseAfterBuffer = true;
        }
    } else {
        this.shouldPlay = false;
    }
};

// Sets the current time. Does not affect play status.
// If the player hasn't loaded yet, it will seek to this time when ready.
Player.prototype.setTime = function(time) {
    this.currentTime = time;
    if (this.youtubePlayerReady) {
        this.currentTime = Math.max(0, Math.min(this.currentTime, this.youtubePlayer.getDuration()));
        this.youtubePlayer.seekTo(this.currentTime);
    }
    this.updateProgress();
};

Player.prototype.jumpToNextMarker = function() {
    var targetMarkerIdx = Math.min((this.currentMarkerIdx === null ? 0 : this.currentMarkerIdx + 1), this.markers.length-1);
    var targetTime = this.markers[targetMarkerIdx].timestamp;
    this.setTime(targetTime);
    this.play();
};

Player.prototype.jumpToPrevMarker = function() {
    var targetMarkerIdx = Math.max(0, (this.currentMarkerIdx === null ? 0 : this.currentMarkerIdx - 1));
    var targetTime = this.markers[targetMarkerIdx].timestamp;
    this.setTime(targetTime);
    this.play();
};

function switchToMobileView(player)
{
    var menuContainerOffset = getElementYOffsetFromPage(titleBar) + parseInt(window.getComputedStyle(titleBar).height);
    if(quotesMenu)
    {
        quotesMenu.previousElementSibling.textContent = '\u{1F5E9}';
        quotesMenu.style.top = menuContainerOffset + "px";
    }
    if(referencesMenu)
    {
        referencesMenu.previousElementSibling.textContent = '\u{1F4D6}';
        referencesMenu.style.top = menuContainerOffset + "px";
    }

    if(filterMenu) { filterMenu.style.top = menuContainerOffset + "px"; }
    if(linkMenu) { linkMenu.style.top = menuContainerOffset + "px"; }

    if(creditsMenu) {
        creditsMenu.previousElementSibling.textContent = '\u{1F46A}';
        creditsMenu.style.top = menuContainerOffset + "px";
    }

    var rightmost = {};
    var markersContainer = player.markersContainer;
    markersContainer.style.height = "auto";
    var episodeMarkerPrev = markersContainer.querySelector(".episodeMarker.prev");
    var episodeMarkerNext = markersContainer.querySelector(".episodeMarker.next");
    var episodeMarkerLast = markersContainer.querySelector(".episodeMarker.last");

    if(episodeMarkerPrev) { episodeMarkerPrev.firstChild.textContent = '\u{23EE}'; }
    if(episodeMarkerNext) { episodeMarkerNext.firstChild.textContent = '\u{23ED}'; rightmost = episodeMarkerNext; }
    else if (episodeMarkerLast) { rightmost = episodeMarkerLast; }

    var markers = markersContainer.querySelector(".markers");

    var controlPrevAnnotation = document.createElement("a");
    controlPrevAnnotation.classList.add("episodeMarker");
    controlPrevAnnotation.classList.add("prevAnnotation");
    controlPrevAnnotation.addEventListener("click", function(ev) {
        player.jumpToPrevMarker();
    });
    var controlPrevAnnotationContent = document.createElement("div");
    controlPrevAnnotationContent.appendChild(document.createTextNode('\u{23F4}'));
    controlPrevAnnotation.appendChild(controlPrevAnnotationContent);

    markersContainer.insertBefore(controlPrevAnnotation, markers);

    var controlNextAnnotation = document.createElement("a");
    controlNextAnnotation.classList.add("episodeMarker");
    controlNextAnnotation.classList.add("nextAnnotation");
    controlNextAnnotation.addEventListener("click", function(ev) {
        player.jumpToNextMarker();
    });
    var controlNextAnnotationContent = document.createElement("div");
    controlNextAnnotationContent.appendChild(document.createTextNode('\u{23F5}'));
    controlNextAnnotation.appendChild(controlNextAnnotationContent);

    if(rightmost)
    {
        markersContainer.insertBefore(controlNextAnnotation, rightmost);
    }
    else
    {
        markersContainer.appendChild(controlNextAnnotation);
    }

    cineraProps.D = devices.MOBILE;
}

function switchToDesktopView(player)
{
    if(quotesMenu)
    {
        quotesMenu.previousElementSibling.textContent = originalTextContent.TitleQuotes;
        quotesMenu.style.top = "100%";
    }
    if(referencesMenu)
    {
        referencesMenu.previousElementSibling.textContent = originalTextContent.TitleReferences;
        referencesMenu.style.top = "100%";
    }
    if(filterMenu) { filterMenu.style.top = "100%"; }
    if(linkMenu) { linkMenu.style.top = "100%"; }
    if(creditsMenu)
    {
        creditsMenu.previousElementSibling.textContent = originalTextContent.TitleCredits;
        creditsMenu.style.top = "100%";
    }

    var markersContainer = player.markersContainer;

    var episodeMarkerPrev = markersContainer.querySelector(".episodeMarker.prev");
    if(episodeMarkerPrev) { episodeMarkerPrev.firstChild.textContent = originalTextContent.EpisodePrev; }
    var episodeMarkerNext = markersContainer.querySelector(".episodeMarker.next");
    if(episodeMarkerNext) { episodeMarkerNext.firstChild.textContent = originalTextContent.EpisodeNext; }

    var prevAnnotation = markersContainer.querySelector(".episodeMarker.prevAnnotation");
    markersContainer.removeChild(prevAnnotation);
    var nextAnnotation = markersContainer.querySelector(".episodeMarker.nextAnnotation");
    markersContainer.removeChild(nextAnnotation);
    cineraProps.D = devices.DESKTOP;
}

// Call this after changing the size of the video container in order to update the youtube player.
Player.prototype.updateSize = function() {
    var width = this.videoContainer.offsetWidth;
    var height = width / 16 * 9;
    if(window.innerHeight > 512 && window.innerWidth > 720)
    {
        if(cineraProps.D == devices.MOBILE)
        {
            switchToDesktopView(this);
        }
        this.markersContainer.style.height = height + "px"; // NOTE(matt): This was the original line here
    }
    else
    {
        if(cineraProps.D == devices.DESKTOP)
        {
            switchToMobileView(this);
        }
    }

    if (this.youtubePlayerReady) {
        this.youtubePlayer.setSize(Math.floor(width), Math.floor(height));
    }
}

// Stops the per-frame work that the player does. Call when you want to hide or get rid of the player.
Player.prototype.halt = function() {
    this.pause();
    this.looping = false;
    if (this.nextFrame) {
        cancelAnimationFrame(this.nextFrame);
        this.nextFrame = null;
    }
}

// Resumes the per-frame work that the player does. Call when you want to show the player again after hiding.
Player.prototype.resume = function() {
    this.looping = true;
    if (!this.nextFrame) {
        this.doFrame();
    }
}

Player.initializeYoutube = function(callback) {
    if (window.APYoutubeAPIReady === undefined) {
        window.APYoutubeAPIReady = false;
        window.APCallbacks = (callback ? [callback] : []);
        window.onYouTubeIframeAPIReady = function() {
            window.APYoutubeAPIReady = true;
            for (var i = 0; i < APCallbacks.length; ++i) {
                APCallbacks[i]();
            }
        };
        var scriptTag = document.createElement("SCRIPT");
        scriptTag.setAttribute("type", "text/javascript");
        scriptTag.setAttribute("src", "https://www.youtube.com/iframe_api");
        document.body.appendChild(scriptTag);
    } else if (window.APYoutubeAPIReady === false) {
        window.APCallbacks.push(callback);
    } else if (window.APYoutubeAPIReady === true) {
        callback();
    }
}

// END PUBLIC INTERFACE

Player.prototype.onMarkerClick = function(marker, ev) {
    var time = marker.timestamp;
    if (this.currentMarker == marker && marker.hoverx !== null) {
        time += (marker.endTime - marker.timestamp) * marker.hoverx;
    }
    this.setTime(time);
    this.play();
};

function getElementXOffsetFromPage(el) {
    var left = 0;
    do {
        left += el.offsetLeft;
    } while (el = el.offsetParent);
    return left;
}

function getElementYOffsetFromPage(el) {
    var top = 0;
    do {
        top += el.offsetTop;
    } while (el = el.offsetParent);
    return top;
}

Player.prototype.onMarkerMouseMove = function(marker, ev) {
    if (this.currentMarker == marker) {
        marker.hoverx = (ev.pageX - getElementXOffsetFromPage(marker.el)) / marker.el.offsetWidth;
    }
};

Player.prototype.onMarkerMouseLeave = function(marker, ev) {
    marker.hoverx = null;
};

Player.prototype.updateProgress = function() {
    var prevMarker = this.currentMarker;
    this.currentMarker = null;
    this.currentMarkerIdx = null;

    for (var i = 0; i < this.markers.length; ++i) {
        var marker = this.markers[i];
        if (marker.timestamp <= this.currentTime && this.currentTime < marker.endTime) {
            this.currentMarker = marker;
            this.currentMarkerIdx = i;
            break;
        }
    }

    if (this.currentMarker) {
        var totalWidth = this.currentMarker.el.offsetWidth;
        var progress = (this.currentTime - this.currentMarker.timestamp) / (this.currentMarker.endTime - this.currentMarker.timestamp);
        if (this.currentMarker.hoverx === null) {
            var pixelWidth = progress * totalWidth;
            this.currentMarker.fadedProgress.style.width = Math.ceil(pixelWidth) + "px";
            this.currentMarker.fadedProgress.style.opacity = pixelWidth - Math.floor(pixelWidth);
            this.currentMarker.progress.style.width = Math.floor(pixelWidth) + "px";
        } else {
            this.currentMarker.fadedProgress.style.opacity = 1;
            this.currentMarker.progress.style.width = Math.floor(Math.min(this.currentMarker.hoverx, progress) * totalWidth) + "px";
            this.currentMarker.fadedProgress.style.width = Math.floor(Math.max(this.currentMarker.hoverx, progress) * totalWidth) + "px";
        }

    }

    if (this.currentMarker != prevMarker) {
        if (prevMarker) {
            prevMarker.el.classList.remove("current");
            prevMarker.fadedProgress.style.width = "0px";
            prevMarker.progress.style.width = "0px";
            prevMarker.hoverx = null;
        }

        if (this.currentMarker) {
            if(this.currentMarkerIdx == this.markers.length - 1)
            {
                localStorage.removeItem(lastAnnotationStorageItem);
            }
            else
            {
                localStorage.setItem(lastAnnotationStorageItem, this.currentMarker.timestamp);
            }
            this.currentMarker.el.classList.add("current");
            this.scrollTo = this.currentMarker.el.offsetTop + this.currentMarker.el.offsetHeight/2.0;
            this.scrollPosition = this.markersContainer.scrollTop;
        }

        if (this.currentMarker) {
            this.refsCallback(this.currentMarker.ref, this.currentMarker.el, this);
        } else if (prevMarker && prevMarker.ref) {
            this.refsCallback(null);
        }
    }
};

Player.prototype.doFrame = function() {
    if (this.playing) {
        this.currentTime = this.youtubePlayer.getCurrentTime();
    }
    this.updateProgress();

    if (this.scrollTo >= 0) {
        var targetPosition = this.scrollTo - this.markersContainer.offsetHeight/2.0;
        targetPosition = Math.max(0, Math.min(targetPosition, this.markersContainer.scrollHeight - this.markersContainer.offsetHeight));
        this.scrollPosition += (targetPosition - this.scrollPosition) * 0.1;
        if (Math.abs(this.scrollPosition - targetPosition) < 1.0) {
            this.markersContainer.scrollTop = targetPosition;
            this.scrollTo = -1;
        } else {
            this.markersContainer.scrollTop = this.scrollPosition;
        }
    }

    this.nextFrame = requestAnimationFrame(this.doFrame.bind(this));
    updateLink();
};

Player.prototype.onYoutubePlayerReady = function() {
    this.youtubePlayerReady = true;
    this.markers[this.markers.length-1].endTime = this.youtubePlayer.getDuration();
    this.updateSize();
    this.youtubePlayer.setPlaybackQuality("hd1080");
    if (this.currentTime > 0) {
        this.currentTime = Math.max(0, Math.min(this.currentTime, this.youtubePlayer.getDuration()));
        this.youtubePlayer.seekTo(this.currentTime, true);
    }
    if (this.shouldPlay) {
        this.youtubePlayer.playVideo();
    }
};

Player.prototype.onYoutubePlayerStateChange = function(ev) {
    if (ev.data == YT.PlayerState.PLAYING) {
        this.playing = true;
        this.currentTime = this.youtubePlayer.getCurrentTime();
    } else {
        this.playing = false;
        if (ev.data == YT.PlayerState.PAUSED || ev.data == YT.PlayerState.BUFFERING) {
            this.currentTime = this.youtubePlayer.getCurrentTime();
            this.updateProgress();
        } else if (ev.data == YT.PlayerState.ENDED) {
            localStorage.removeItem(lastAnnotationStorageItem);
            this.currentTime = null;
            this.updateProgress();
        }
    }

    this.buffering = ev.data == YT.PlayerState.BUFFERING;
    if (this.playing && this.pauseAfterBuffer) {
        this.pauseAfterBuffering = false;
        this.pause();
    }
};

Player.prototype.onYoutubePlayerPlaybackRateChange = function(ev) {
    this.speed = ev.data;
};

Player.prototype.onYoutubeReady = function() {
    var youtubePlayerDiv = document.createElement("DIV");
    youtubePlayerDiv.id = "youtube_player_" + Player.youtubePlayerCount++;
    this.videoContainer.appendChild(youtubePlayerDiv);
    this.youtubePlayer = new YT.Player(youtubePlayerDiv.id, {
        videoId: this.videoContainer.getAttribute("data-videoId"),
        width: this.videoContainer.offsetWidth,
        height: this.videoContainer.offsetWidth / 16 * 9,
        //playerVars: { disablekb: 1 },
        events: {
            "onReady": this.onYoutubePlayerReady.bind(this),
            "onStateChange": this.onYoutubePlayerStateChange.bind(this),
            "onPlaybackRateChange": this.onYoutubePlayerPlaybackRateChange.bind(this)
        }
    });
};

Player.youtubePlayerCount = 0;

// NOTE(matt): Hereafter is my stuff. Beware!

function toggleFilterMode() {
    if(filterMode == "inclusive")
    {
        filterModeElement.classList.remove("inclusive");
        filterModeElement.classList.add("exclusive");
        filterMode = "exclusive";
    }
    else
    {
        filterModeElement.classList.remove("exclusive");
        filterModeElement.classList.add("inclusive");
        filterMode = "inclusive";
    }
    applyFilter();
}

function updateLink()
{
    if(link && player)
    {
        if(linkAnnotation == true)
        {
            if(player.currentMarker)
            {
                link.value = baseURL + "#" + player.currentMarker.timestamp;
            }
            else
            {
                link.value = baseURL;
            }
        }
        else
        {
            link.value = baseURL + "#" + Math.round(player.youtubePlayer.getCurrentTime());
        }
    }
}

function toggleLinkMode(linkMode, link)
{
    linkAnnotation = !linkAnnotation;
    if(linkAnnotation == true)
    {
        linkMode.textContent = "Link to: current timestamp";
    }
    else
    {
        linkMode.textContent = "Link to: nearest second";
    }
    updateLink();
}

function toggleFilterOrLinkMode()
{
    for(menuIndex in menuState)
    {
        if(menuState[menuIndex].classList.contains("filter_container") && menuState[menuIndex].classList.contains("visible"))
        {
            toggleFilterMode();
        }
        if(menuState[menuIndex].classList.contains("link_container") && menuState[menuIndex].classList.contains("visible"))
        {
            toggleLinkMode(linkMode, link);
        }
    }
}

function toggleMenuVisibility(element) {
    if(element.classList.contains("visible"))
    {
        element.classList.remove("visible");
        element.parentNode.classList.remove("visible");
        if(focusedElement)
        {
            focusedElement.classList.remove("focused");
            focusedElement = null;
        }
        if(focusedIdentifier)
        {
            focusedIdentifier.classList.remove("focused");
            focusedIdentifier = null;
        }
    }
    else
    {
        for(menuIndex in menuState)
        {
            menuState[menuIndex].classList.remove("visible");
            menuState[menuIndex].parentNode.classList.remove("visible");
            if(focusedElement)
            {
                focusedElement.classList.remove("focused");
            }
            if(focusedIdentifier)
            {
                focusedIdentifier.classList.remove("focused");
            }
        }
        element.classList.add("visible");
        element.parentNode.classList.add("visible");

        if(element.classList.contains("quotes_container"))
        {
            if(!lastFocusedQuote)
            {
                lastFocusedQuote = element.querySelectorAll(".ref")[0];
            }
            focusedElement = lastFocusedQuote;
            focusedElement.classList.add("focused");
        }
        else if(element.classList.contains("references_container"))
        {
            if(!lastFocusedReference || !lastFocusedIdentifier)
            {
                lastFocusedReference = element.querySelectorAll(".ref")[0];
                lastFocusedIdentifier = lastFocusedReference.querySelector(".ref_indices").firstElementChild;
            }
            focusedElement = lastFocusedReference;
            focusedElement.classList.add("focused");
            focusedIdentifier = lastFocusedIdentifier;
            focusedIdentifier.classList.add("focused");
        }
        else if(element.classList.contains("filter_container"))
        {
            if(!lastFocusedCategory)
            {
                lastFocusedCategory = element.querySelectorAll(".filter_content")[0];
            }
            focusedElement = lastFocusedCategory;
            focusedElement.classList.add("focused");
        }
        else if(element.classList.contains("credits_container"))
        {
            if(!lastFocusedCreditItem)
            {
                if(element.querySelectorAll(".credit .person")[0].nextElementSibling)
                {
                    lastFocusedCreditItem = element.querySelectorAll(".credit .support")[0];
                    focusedElement = lastFocusedCreditItem;
                    focusedElement.classList.add("focused");
                    setSpriteLightness(focusedElement.firstChild);
                }
                else
                {
                    lastFocusedCreditItem = element.querySelectorAll(".credit .person")[0];
                    focusedElement = lastFocusedCreditItem;
                    focusedElement.classList.add("focused");
                }
            }
            else
            {
                focusedElement = lastFocusedCreditItem;
                focusedElement.classList.add("focused");
            }
        }
    }
}

function handleMouseOverViewsMenu()
{
    switch(cineraProps.V)
    {
        case views.REGULAR:
        case views.THEATRE:
            {
                viewsContainer.style.display = "block";
            } break;
        case views.SUPERTHEATRE:
            {
                viewsContainer.style.display = "none";
            } break;
    }
}

function enterFullScreen_()
{
    if(!document.mozFullScreen && !document.webkitFullScreen)
    {
        if(cinera.mozRequestFullScreen)
        {
            cinera.mozRequestFullScreen();
        }
        else
        {
            cinera.webkitRequestFullScreen(Element.ALLOW_KEYBOARD_INPUT);
        }
    }
}

function leaveFullScreen_()
{
    if(document.mozCancelFullScreen)
    {
        document.mozCancelFullScreen();
    }
    else
    {
        document.webkitExitFullscreen();
    }
}

function toggleTheatreMode() {
    switch(cineraProps.V)
    {
        case views.REGULAR:
            {
                cineraProps.C = cinera.style.backgroundColor;
                cineraProps.Z = cinera.style.zIndex;
                cineraProps.X = cinera.style.left;
                cineraProps.Y = cinera.style.top;
                cineraProps.W = cinera.style.width;
                cineraProps.mW = cinera.style.maxWidth;
                cineraProps.H = cinera.style.height;
                cineraProps.mH = cinera.style.maxHeight;
                cineraProps.P = cinera.style.position;

                cinera.style.backgroundColor = "#000";
                cinera.style.zIndex = 64;
                cinera.style.left = 0;
                cinera.style.top = 0;
                cinera.style.width = "100%";
                cinera.style.maxWidth = "100%";
                cinera.style.height = "100%";
                cinera.style.maxHeight = "100%";
                cinera.style.position = "fixed";

                viewItems[0].setAttribute("data-id", "regular");
                viewItems[0].setAttribute("title", "Regular mode");
                viewItems[0].firstChild.nodeValue = "📺";
            } cineraProps.V = views.THEATRE; localStorage.setItem(cineraViewStorageItem, views.THEATRE); break;
        case views.SUPERTHEATRE:
            {
                leaveFullScreen_();
            }
        case views.THEATRE:
            {
                cinera.style.backgroundColor = cineraProps.C;
                cinera.style.zIndex = cineraProps.Z;
                cinera.style.left = cineraProps.X;
                cinera.style.top = cineraProps.Y;
                cinera.style.width = cineraProps.W;
                cinera.style.maxWidth = cineraProps.mW;
                cinera.style.height = cineraProps.H;
                cinera.style.maxHeight = cineraProps.mH;
                cinera.style.position = cineraProps.P;

                viewItems[0].setAttribute("data-id", "theatre");
                viewItems[0].setAttribute("title", "Theatre mode");
                viewItems[0].firstChild.nodeValue = "🎭";
            } cineraProps.V = views.REGULAR; localStorage.removeItem(cineraViewStorageItem); break;
    }
    player.updateSize();
}

function toggleSuperTheatreMode()
{
    switch(cineraProps.V)
    {
        case views.REGULAR:
            {
                toggleTheatreMode();
            }
        case views.THEATRE:
            {
                enterFullScreen_();
            } cineraProps.V = views.SUPERTHEATRE; localStorage.setItem(cineraViewStorageItem, views.SUPERTHEATRE); break;
        case views.SUPERTHEATRE:
            {
                leaveFullScreen_();
                toggleTheatreMode();
            } cineraProps.V = views.REGULAR; localStorage.removeItem(cineraViewStorageItem); break;
    }
    player.updateSize();
}

function AscribeTemporaryResponsibility(Element, Milliseconds)
{
    if(!Element.classList.contains("responsible"))
    {
        Element.classList.add("responsible");
    }
    setTimeout(function() { Element.classList.remove("responsible"); }, Milliseconds);
}

function SelectText(inputElement)
{
    inputElement.select();
}

function CopyToClipboard(inputElement)
{
    SelectText(inputElement);
    document.execCommand("copy");
    AscribeTemporaryResponsibility(linkMenu.parentNode, 8000);
}

function handleKey(key) {
    var gotKey = true;
    switch (key) {
        case "q": {
            if(quotesMenu)
            {
                toggleMenuVisibility(quotesMenu)
            }
        } break;
        case "r": {
            if(referencesMenu)
            {
                toggleMenuVisibility(referencesMenu)
            }
        } break;
        case "f": {
            if(filterMenu)
            {
                toggleMenuVisibility(filterMenu)
            }
        } break;
        case "y": {
            if(linkMenu)
            {
                toggleMenuVisibility(linkMenu)
            }
            break;
        }
        case "c": {
            if(creditsMenu)
            {
                toggleMenuVisibility(creditsMenu)
            }
        } break;
        case "t": {
            if(cinera)
            {
                toggleTheatreMode();
            }
        } break;
        case "T": {
            if(cinera)
            {
                toggleSuperTheatreMode();
            }
        } break;

        case "Enter": {
            if(focusedElement)
            {
                if(focusedElement.parentNode.classList.contains("quotes_container"))
                {
                    var time = focusedElement.querySelector(".timecode").getAttribute("data-timestamp");
                    player.setTime(parseInt(time, 10));
                    player.play();
                }
                else if(focusedElement.parentNode.classList.contains("references_container"))
                {
                    var time = focusedIdentifier.getAttribute("data-timestamp");
                    player.setTime(parseInt(time, 10));
                    player.play();
                }
                else if(focusedElement.parentNode.classList.contains("credit"))
                {
                    if(focusedElement.hasAttribute)
                    {
                        var url = focusedElement.getAttribute("href");
                        window.open(url, "_blank");
                    }
                }
            }
            else
            {
                console.log("TODO(matt): Implement me, perhaps?\n");
            }
        } break;

        case "o": {
            if(focusedElement)
            {
                if(focusedElement.parentNode.classList.contains("references_container") ||
                    focusedElement.parentNode.classList.contains("quotes_container"))
                {
                    var url = focusedElement.getAttribute("href");
                    window.open(url, "_blank");
                }
                else if(focusedElement.parentNode.classList.contains("credit"))
                {
                    if(focusedElement.hasAttribute("href"))
                    {
                        var url = focusedElement.getAttribute("href");
                        window.open(url, "_blank");
                    }
                }
            }
        } break;

        case "w": case "k": case "ArrowUp": {
            if(focusedElement)
            {
                if(focusedElement.parentNode.classList.contains("quotes_container"))
                {
                    if(focusedElement.previousElementSibling)
                    {
                        focusedElement.classList.remove("focused");

                        lastFocusedQuote = focusedElement.previousElementSibling;
                        focusedElement = lastFocusedQuote;
                        focusedElement.classList.add("focused");
                    }
                }
                else if(focusedElement.parentNode.classList.contains("references_container"))
                {
                    if(focusedElement.previousElementSibling)
                    {
                        focusedElement.classList.remove("focused");
                        focusedIdentifier.classList.remove("focused");

                        lastFocusedReference = focusedElement.previousElementSibling;
                        focusedElement = lastFocusedReference;
                        focusedElement.classList.add("focused");

                        lastFocusedIdentifier = focusedElement.querySelector(".ref_indices").firstElementChild;
                        focusedIdentifier = lastFocusedIdentifier;
                        focusedIdentifier.classList.add("focused");
                    }
                }
                else if(focusedElement.parentNode.parentNode.classList.contains("filters"))
                {
                    if(focusedElement.previousElementSibling &&
                        focusedElement.previousElementSibling.classList.contains("filter_content"))
                    {
                        focusedElement.classList.remove("focused");

                        lastFocusedCategory = focusedElement.previousElementSibling;
                        focusedElement = lastFocusedCategory;
                        focusedElement.classList.add("focused");
                    }
                }
                else if(focusedElement.parentNode.classList.contains("credit"))
                {
                    if(focusedElement.parentNode.previousElementSibling)
                    {
                        focusedElement.classList.remove("focused");
                        if(focusedElement.parentNode.previousElementSibling.querySelector(".support") &&
                            focusedElement.classList.contains("support"))
                        {
                            setSpriteLightness(focusedElement.firstChild);
                            lastFocusedCreditItem = focusedElement.parentNode.previousElementSibling.querySelector(".support");
                            focusedElement = lastFocusedCreditItem;
                            focusedElement.classList.add("focused");
                            setSpriteLightness(focusedElement.firstChild);
                        }
                        else
                        {
                            lastFocusedCreditItem = focusedElement.parentNode.previousElementSibling.querySelector(".person");
                            focusedElement = lastFocusedCreditItem;
                            focusedElement.classList.add("focused");
                        }
                    }
                }
            }
        } break;

        case "s": case "j": case "ArrowDown": {
            if(focusedElement)
            {
                if(focusedElement.parentNode.classList.contains("quotes_container"))
                {
                    if(focusedElement.nextElementSibling)
                    {
                        focusedElement.classList.remove("focused");

                        lastFocusedQuote = focusedElement.nextElementSibling;
                        focusedElement = lastFocusedQuote;
                        focusedElement.classList.add("focused");
                    }
                }
                else if(focusedElement.parentNode.classList.contains("references_container"))
                {
                    if(focusedElement.nextElementSibling)
                    {
                        focusedElement.classList.remove("focused");
                        focusedIdentifier.classList.remove("focused");

                        lastFocusedReference = focusedElement.nextElementSibling;
                        focusedElement = lastFocusedReference;
                        focusedElement.classList.add("focused");

                        lastFocusedIdentifier = focusedElement.querySelector(".ref_indices").firstElementChild;
                        focusedIdentifier = lastFocusedIdentifier;
                        focusedIdentifier.classList.add("focused");
                    }
                }
                else if(focusedElement.parentNode.parentNode.classList.contains("filters"))
                {
                    if(focusedElement.nextElementSibling &&
                        focusedElement.nextElementSibling.classList.contains("filter_content"))
                    {
                        focusedElement.classList.remove("focused");

                        lastFocusedCategory = focusedElement.nextElementSibling;
                        focusedElement = lastFocusedCategory;
                        focusedElement.classList.add("focused");
                    }
                }
                else if(focusedElement.parentNode.classList.contains("credit"))
                {
                    if(focusedElement.parentNode.nextElementSibling)
                    {
                        focusedElement.classList.remove("focused");
                        if(focusedElement.parentNode.nextElementSibling.querySelector(".support") &&
                            focusedElement.classList.contains("support"))
                        {
                            setSpriteLightness(focusedElement.firstChild);
                            lastFocusedCreditItem = focusedElement.parentNode.nextElementSibling.querySelector(".support");
                            focusedElement = lastFocusedCreditItem;
                            focusedElement.classList.add("focused");
                            setSpriteLightness(focusedElement.firstChild);
                        }
                        else
                        {
                            lastFocusedCreditItem = focusedElement.parentNode.nextElementSibling.querySelector(".person");
                            focusedElement = lastFocusedCreditItem;
                            focusedElement.classList.add("focused");
                        }
                    }
                }
            }
        } break;

        case "a": case "h": case "ArrowLeft": {
            if(focusedElement)
            {
                if(focusedElement.parentNode.classList.contains("references_container"))
                {
                    if(focusedIdentifier.previousElementSibling)
                    {
                        focusedIdentifier.classList.remove("focused");
                        lastFocusedIdentifier = focusedIdentifier.previousElementSibling;
                        focusedIdentifier = lastFocusedIdentifier;
                        focusedIdentifier.classList.add("focused");
                    }
                    else if(focusedIdentifier.parentNode.previousElementSibling.classList.contains("ref_indices"))
                    {
                        focusedIdentifier.classList.remove("focused");
                        lastFocusedIdentifier = focusedIdentifier.parentNode.previousElementSibling.lastElementChild;
                        focusedIdentifier = lastFocusedIdentifier;
                        focusedIdentifier.classList.add("focused");
                    }
                }
                else if(focusedElement.classList.contains("filter_content"))
                {
                    if(focusedElement.parentNode.classList.contains("filter_media") &&
                        focusedElement.parentNode.previousElementSibling)
                    {
                        focusedElement.classList.remove("focused");
                        lastFocusedMedium = focusedElement;

                        if(!lastFocusedTopic)
                        {
                            lastFocusedTopic = focusedElement.parentNode.previousElementSibling.children[1];
                        }
                        lastFocusedCategory = lastFocusedTopic;
                        focusedElement = lastFocusedCategory;
                        focusedElement.classList.add("focused");
                    }
                }
                else if(focusedElement.parentNode.classList.contains("credit"))
                {
                    if(focusedElement.classList.contains("support"))
                    {
                        focusedElement.classList.remove("focused");

                        lastFocusedCreditItem = focusedElement.previousElementSibling;
                        if(focusedElement.firstChild.classList.contains("cineraSprite"))
                        {
                            setSpriteLightness(focusedElement.firstChild);
                        }
                        focusedElement = lastFocusedCreditItem;
                        focusedElement.classList.add("focused");
                    }
                }
            }
        } break;

        case "d": case "l": case "ArrowRight": {
            if(focusedElement)
            {
                if(focusedElement.parentNode.classList.contains("references_container"))
                {
                    if(focusedIdentifier.nextElementSibling)
                    {
                        focusedIdentifier.classList.remove("focused");

                        lastFocusedIdentifier = focusedIdentifier.nextElementSibling;
                        focusedIdentifier = lastFocusedIdentifier;
                        focusedIdentifier.classList.add("focused");
                    }
                    else if(focusedIdentifier.parentNode.nextElementSibling)
                    {
                        focusedIdentifier.classList.remove("focused");
                        lastFocusedIdentifier = focusedIdentifier.parentNode.nextElementSibling.firstElementChild;
                        focusedIdentifier = lastFocusedIdentifier;
                        focusedIdentifier.classList.add("focused");
                    }
                }
                else if(focusedElement.classList.contains("filter_content"))
                {
                    if(focusedElement.parentNode.classList.contains("filter_topics") &&
                        focusedElement.parentNode.nextElementSibling)
                    {
                        focusedElement.classList.remove("focused");
                        lastFocusedTopic = focusedElement;

                        if(!lastFocusedMedium)
                        {
                            lastFocusedMedium = focusedElement.parentNode.nextElementSibling.children[1];
                        }
                        lastFocusedCategory = lastFocusedMedium;
                        focusedElement = lastFocusedCategory;
                        focusedElement.classList.add("focused");
                    }
                }
                else if(focusedElement.parentNode.classList.contains("credit"))
                {
                    if(focusedElement.classList.contains("person") &&
                        focusedElement.nextElementSibling)
                    {
                        focusedElement.classList.remove("focused");

                        lastFocusedCreditItem = focusedElement.nextElementSibling;
                        focusedElement = lastFocusedCreditItem;
                        focusedElement.classList.add("focused");
                        if(focusedElement.firstChild.classList.contains("cineraSprite"))
                        {
                            setSpriteLightness(focusedElement.firstChild);
                        }
                    }
                }
            }
        } break;

        case "x": case " ": {
            if(focusedElement && focusedElement.classList.contains("filter_content"))
            {
                filterItemToggle(focusedElement);
                if(focusedElement.nextElementSibling &&
                    focusedElement.nextElementSibling.classList.contains("filter_content"))
                {
                    focusedElement.classList.remove("focused");
                    if(focusedElement.parentNode.classList.contains("filter_topics"))
                    {
                        lastFocusedTopic = focusedElement.nextElementSibling;
                        lastFocusedCategory = lastFocusedTopic;
                    }
                    else
                    {
                        lastFocusedMedium = focusedElement.nextElementSibling;
                        lastFocusedCategory = lastFocusedMedium;
                    }
                    lastFocusedElement = lastFocusedCategory;
                    focusedElement = lastFocusedElement;
                    focusedElement.classList.add("focused");
                }
            }
        } break;

        case "X": case "capitalSpace": {
            if(focusedElement && focusedElement.classList.contains("filter_content"))
            {
                filterItemToggle(focusedElement);
                if(focusedElement.previousElementSibling &&
                    focusedElement.previousElementSibling.classList.contains("filter_content"))
                {
                    focusedElement.classList.remove("focused");
                    if(focusedElement.parentNode.classList.contains("filter_topics"))
                    {
                        lastFocusedTopic = focusedElement.previousElementSibling;
                        lastFocusedCategory = lastFocusedTopic;
                    }
                    else
                    {
                        lastFocusedMedium = focusedElement.previousElementSibling;
                        lastFocusedCategory = lastFocusedMedium;
                    }
                    lastFocusedElement = lastFocusedCategory;
                    focusedElement = lastFocusedElement;
                    focusedElement.classList.add("focused");
                }
            }
        } break;

        case "z": {
            toggleFilterOrLinkMode();
        } break;

        case "v": {
            if(focusedElement && focusedElement.classList.contains("filter_content"))
            {
                invertFilter(focusedElement)
            }
        } break;

        case "V": {
            resetFilter();
        } break;

        case "?": {
            helpDocumentation.classList.toggle("visible");
        } break;

        case 'N':
        case 'J':
        case 'S': {
            player.jumpToNextMarker();
        } break;

        case 'P':
        case 'K':
        case 'W': {
            player.jumpToPrevMarker();
        } break;
        case '[':
        case '<': {
            if(prevEpisode)
            {
                location = prevEpisode.href;
            }
        } break;
        case ']':
        case '>': {
            if(nextEpisode)
            {
                location = nextEpisode.href;
            }
        } break;
        case 'Y': {
            if(cineraLink)
            {
                if(linkAnnotation == false && player.playing)
                {
                    player.pause();
                }
                if(linkMenu && !linkMenu.classList.contains("visible"))
                {
                    toggleMenuVisibility(linkMenu);
                }
                SelectText(cineraLink);
            }
        }
        default: {
            gotKey = false;
        } break;
    }
    return gotKey;
}

function applyFilter() {
    if(filterMode == "exclusive")
    {
        for(var i = 0; i < testMarkers.length; ++i)
        {
            var testCategories = testMarkers[i].classList;
            for(var j = 0; j < testCategories.length; ++j)
            {
                if((testCategories[j].startsWith("off_")) && !testMarkers[i].classList.contains("skip"))
                {
                    testMarkers[i].classList.add("skip");
                }
            }
        }
    }
    else
    {
        for(var i = 0; i < testMarkers.length; ++i)
        {
            var testCategories = testMarkers[i].classList;
            for(var j = 0; j < testCategories.length; ++j)
            {
                if((testCategories[j] in filterState || testCategories[j].startsWith("cat_")) && testMarkers[i].classList.contains("skip"))
                {
                    testMarkers[i].classList.remove("skip");
                }
            }
        }
    }
}

function filterItemToggle(filterItem) {
    var selectedCategory = filterItem.classList[1];
    filterState[selectedCategory].off = !filterState[selectedCategory].off;

    if(filterState[selectedCategory].off)
    {
        filterItem.classList.add("off");
        disableSprite(filterItem);
        if(!filterItem.parentNode.classList.contains("filter_media"))
        {
            filterItem.querySelector(".icon").style.backgroundColor = "transparent";
        }
        var testMarkers = playerContainer.querySelectorAll(".marker." + selectedCategory + ", .marker.cat_" + selectedCategory);
        for(var j = 0; j < testMarkers.length; ++j)
        {
            if(filterState[selectedCategory].type == "topic")
            {
                testMarkers[j].classList.remove("cat_" + selectedCategory);
                testMarkers[j].classList.add("off_" + selectedCategory);
                var markerCategories = testMarkers[j].querySelectorAll(".category." + selectedCategory);
                for(var k = 0; k < markerCategories.length; ++k)
                {
                    if(markerCategories[k].classList.contains(selectedCategory))
                    {
                        markerCategories[k].classList.add("off");
                        markerCategories[k].style.backgroundColor = "transparent";
                    }
                }
            }
            else
            {
                var markerCategories = testMarkers[j].querySelectorAll(".categoryMedium." + selectedCategory);
                for(var k = 0; k < markerCategories.length; ++k)
                {
                    if(markerCategories[k].classList.contains(selectedCategory))
                    {
                        markerCategories[k].classList.add("off");
                        disableSprite(markerCategories[k]);
                    }
                }
                testMarkers[j].classList.remove(selectedCategory);
                testMarkers[j].classList.add("off_" + selectedCategory);
            }

            Skipping = 1;
            if(filterMode == "exclusive")
            {
                testMarkers[j].classList.add("skip");
            }
            else
            {
                var markerClasses = testMarkers[j].classList;
                for(var k = 0; k < markerClasses.length; ++k)
                {
                    if(markerClasses[k] in filterState || markerClasses[k].replace(/^cat_/, "") in filterState)
                    {
                        Skipping = 0;
                    }
                }
                if(Skipping)
                {
                    testMarkers[j].classList.add("skip");
                }
            }

        }
    }
    else
    {
        filterItem.classList.remove("off");
        enableSprite(filterItem);
        if(!filterItem.parentNode.classList.contains("filter_media"))
        {
            filterItem.querySelector(".icon").style.backgroundColor = getComputedStyle(filterItem.querySelector(".icon")).getPropertyValue("border-color");
        }
        setDotLightness(filterItem.querySelector(".icon"));
        var testMarkers = document.querySelectorAll(".marker.off_" + selectedCategory);
        for(var j = 0; j < testMarkers.length; ++j)
        {
            if(filterState[selectedCategory].type == "topic")
            {
                testMarkers[j].classList.remove("off_" + selectedCategory);
                testMarkers[j].classList.add("cat_" + selectedCategory);
                var markerCategories = testMarkers[j].querySelectorAll(".category." + selectedCategory);
                for(var k = 0; k < markerCategories.length; ++k)
                {
                    if(markerCategories[k].classList.contains(selectedCategory))
                    {
                        markerCategories[k].classList.remove("off");
                        markerCategories[k].style.backgroundColor = getComputedStyle(markerCategories[k]).getPropertyValue("border-color");
                        setDotLightness(markerCategories[k]);
                    }
                }
            }
            else
            {
                testMarkers[j].classList.remove("off_" + selectedCategory);
                testMarkers[j].classList.add(selectedCategory);
                var markerCategories = testMarkers[j].querySelectorAll(".categoryMedium." + selectedCategory);
                for(var k = 0; k < markerCategories.length; ++k)
                {
                    if(markerCategories[k].classList.contains(selectedCategory))
                    {
                        markerCategories[k].classList.remove("off");
                        enableSprite(markerCategories[k]);
                    }
                }
            }

            Skipping = 0;
            if(filterMode == "inclusive")
            {
                testMarkers[j].classList.remove("skip");
            }
            else
            {
                var markerClasses = testMarkers[j].classList;
                for(var k = 0; k < markerClasses.length; ++k)
                {
                    if(markerClasses[k].startsWith("off_"))
                    {
                        Skipping = 1;
                    }
                }
                if(!Skipping)
                {
                    testMarkers[j].classList.remove("skip");
                }
            }
        }
    }
}

function resetFilter() {
    for(i in filterItems)
    {
        if(filterItems[i].classList)
        {
            var selectedCategory = filterItems[i].classList[1];
            if(filterInitState[selectedCategory].off ^ filterState[selectedCategory].off)
            {
                filterItemToggle(filterItems[i]);
            }
        }
    }

    if(filterMode == "inclusive")
    {
        toggleFilterMode();
    }
}

function invertFilter(focusedElement) {
    var siblings = focusedElement.parentNode.querySelectorAll(".filter_content");
    for(i in siblings)
    {
        if(siblings[i].classList)
        {
            filterItemToggle(siblings[i]);
        }
    }
}

function resetFade() {
    filter.classList.remove("responsible");
    filter.querySelector(".filter_mode").classList.remove("responsible");
    var responsibleCategories = filter.querySelectorAll(".filter_content.responsible");
    for(var i = 0; i < responsibleCategories.length; ++i)
    {
        responsibleCategories[i].classList.remove("responsible");
    }
}

function onRefChanged(ref, element, player) {
    if(element.classList.contains("skip"))
    {
        var ErrorCount = 0;
        if(!filter) { console.log("Missing filter_container div"); ErrorCount++; }
        if(!filterState) { console.log("Missing filterState object"); ErrorCount++; }
        if(ErrorCount > 0)
        {
            switch(ErrorCount)
            {
                case 1:
                    { console.log("This should have been generated by Cinera along with the following element containing the \"skip\" class:"); } break;
                default:
                    { console.log("These should have been generated by Cinera along with the following element containing the \"skip\" class:"); } break;
            }
            console.log(element); return;
        }

        if(!filter.classList.contains("responsible"))
        {
            filter.classList.add("responsible");
        }

        for(var selector = 0; selector < element.classList.length; ++selector)
        {
            if(element.classList[selector].startsWith("off_"))
            {
                if(!filter.querySelector(".filter_content." + element.classList[selector].replace(/^off_/, "")).classList.contains("responsible"))
                {
                    filter.querySelector(".filter_content." + element.classList[selector].replace(/^off_/, "")).classList.add("responsible");
                }
            }
            if(element.classList[selector].startsWith("cat_") || element.classList[selector] in filterState)
            {
                if(!filter.querySelector(".filter_mode").classList.add("responsible"))
                {
                    filter.querySelector(".filter_mode").classList.add("responsible");
                }
            }
            setTimeout(resetFade, 8000);
        }
        if(player && player.playing)
        {
            player.jumpToNextMarker();
        }
        return;
    }

    for (var MenuIndex = 0; MenuIndex < sourceMenus.length; ++MenuIndex)
    {
        var SetMenu = 0;
        if (ref !== undefined && ref !== null) {
            var refElements = sourceMenus[MenuIndex].querySelectorAll(".refs .ref");
            var refs = ref.split(",");

            for (var i = 0; i < refElements.length; ++i) {
                if (refs.includes(refElements[i].getAttribute("data-id"))) {
                    refElements[i].classList.add("current");
                    SetMenu = 1;
                } else {
                    refElements[i].classList.remove("current");
                }
            }
            if(SetMenu) {
                sourceMenus[MenuIndex].classList.add("current");
            } else {
                sourceMenus[MenuIndex].classList.remove("current");
            }

        } else {
            sourceMenus[MenuIndex].classList.remove("current");
            var refs = sourceMenus[MenuIndex].querySelectorAll(".refs .ref");
            for (var i = 0; i < refs.length; ++i) {
                refs[i].classList.remove("current");
            }
        }
    }
}

function navigateFilter(filterItem) {
    if(filterItem != lastFocusedCategory)
    {
        lastFocusedCategory.classList.remove("focused");
        unfocusSprite(lastFocusedCategory);
        if(filterItem.parentNode.classList.contains("filter_topics"))
        {
            lastFocusedTopic = filterItem;
            lastFocusedCategory = lastFocusedTopic;
        }
        else
        {
            lastFocusedMedium = filterItem;
            lastFocusedCategory = lastFocusedMedium;
        }
        focusedElement = lastFocusedCategory;
        focusedElement.classList.add("focused");
        focusSprite(focusedElement);
    }
}

function mouseOverQuotes(quote) {
    if(focusedElement && quote != lastFocusedQuote)
    {
        focusedElement.classList.remove("focused");
        lastFocusedQuote = quote;
        focusedElement = lastFocusedQuote;
        focusedElement.classList.add("focused");
    }
}

function mouseOverReferences(reference) {
    if(focusedElement && reference != lastFocusedReference)
    {
        focusedElement.classList.remove("focused")
        lastFocusedReference = reference;
    }
    focusedElement = lastFocusedReference;
    focusedElement.classList.add("focused");

    var ourIdentifiers = reference.querySelectorAll(".timecode");
    weWereLastFocused = false;
    for(var k = 0; k < ourIdentifiers.length; ++k)
    {
        if(ourIdentifiers[k] == lastFocusedIdentifier)
        {
            weWereLastFocused = true;
        }
    }
    if(!weWereLastFocused)
    {
        lastFocusedIdentifier.classList.remove("focused");
        lastFocusedIdentifier = ourIdentifiers[0];
    }
    focusedIdentifer = lastFocusedIdentifier;
    focusedIdentifer.classList.add("focused");

    for(var l = 0; l < ourIdentifiers.length; ++l)
    {
        ourIdentifiers[l].addEventListener("mouseenter", function(ev) {
            if(this != lastFocusedIdentifier)
            {
                lastFocusedIdentifier.classList.remove("focused");
                lastFocusedIdentifier = this;
                lastFocusedIdentifier.classList.add("focused");
            }
        })
    }
}

function mouseSkipToTimecode(player, time, ev)
{
    player.setTime(parseInt(time, 10));
    player.play();
    ev.preventDefault();
    ev.stopPropagation();
    return false;
}

function handleMouseOverMenu(menu, eventType)
{
    if(!(menu.classList.contains("visible")) && eventType == "mouseenter" ||
        menu.classList.contains("visible") && eventType == "mouseleave")
    {
        if(menu.classList.contains("quotes"))
        {
            toggleMenuVisibility(quotesMenu);
        }
        else if(menu.classList.contains("references"))
        {
            toggleMenuVisibility(referencesMenu);
        }
        else if(menu.classList.contains("filter"))
        {
            toggleMenuVisibility(filterMenu);
        }
        else if(menu.classList.contains("link"))
        {
            toggleMenuVisibility(linkMenu);
        }
        else if(menu.classList.contains("credits"))
        {
            toggleMenuVisibility(creditsMenu);
        }
    }
    if(eventType == "click" && menu.classList.contains("help"))
    {
        helpDocumentation.classList.toggle("visible");
    }
}

function RGBtoHSL(colour)
{
	var rgb = colour.slice(4, -1).split(", ");
    var red = rgb[0];
    var green = rgb[1];
    var blue = rgb[2];
    var min = Math.min(red, green, blue);
    var max = Math.max(red, green, blue);
    var chroma = max - min;
    var hue = 0;
    if(max == red)
    {
        hue = ((green - blue) / chroma) % 6;
    }
    else if(max == green)
    {
        hue = ((blue - red) / chroma) + 2;
    }
    else if(max == blue)
    {
        hue = ((red - green) / chroma) + 4;
    }

    var saturation = chroma / 255 * 100;
    hue = (hue * 60) < 0 ? 360 + (hue * 60) : (hue * 60);

    return [hue, saturation]
}

function getBackgroundBrightness(element) {
    var colour = getComputedStyle(element).getPropertyValue("background-color");
    var depth = 0;
    while((colour == "transparent" || colour == "rgba(0, 0, 0, 0)") && depth <= 4)
    {
        element = element.parentNode;
        colour = getComputedStyle(element).getPropertyValue("background-color");
        ++depth;
    }
	var rgb = colour.slice(4, -1).split(", ");
	var result = Math.sqrt(rgb[0] * rgb[0] * .241 +
	rgb[1] * rgb[1] * .691 +
	rgb[2] * rgb[2] * .068);
    console.log(result);
    return result;
}

function setTextLightness(textElement)
{
    var textHue = textElement.getAttribute("data-hue");
    var textSaturation = textElement.getAttribute("data-saturation");
    if(getBackgroundBrightness(textElement.parentNode) < 127)
    {
        textElement.style.color = ("hsl(" + textHue + ", " + textSaturation + ", 76%)");
    }
    else
    {
        textElement.style.color = ("hsl(" + textHue + ", " + textSaturation + ", 24%)");
    }
}

function setDotLightness(topicDot)
{
    var Hue = RGBtoHSL(getComputedStyle(topicDot).getPropertyValue("background-color"))[0];
    var Saturation = RGBtoHSL(getComputedStyle(topicDot).getPropertyValue("background-color"))[1];
    if(getBackgroundBrightness(topicDot.parentNode) < 127)
    {
        topicDot.style.backgroundColor = ("hsl(" + Hue + ", " + Saturation + "%, 76%)");
        topicDot.style.borderColor = ("hsl(" + Hue + ", " + Saturation + "%, 76%)");
    }
    else
    {
        topicDot.style.backgroundColor = ("hsl(" + Hue + ", " + Saturation + "%, 47%)");
        topicDot.style.borderColor = ("hsl(" + Hue + ", " + Saturation + "%, 47%)");
    }
}
