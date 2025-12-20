document.body.style.overflowY = "scroll";

if (location.hash && location.hash.length > 0) {
    var initialQuery = location.hash;
    if (initialQuery[0] == "#") {
        initialQuery = initialQuery.slice(1);
    }
    document.getElementById("query").value = decodeURIComponent(initialQuery);
}

var indexControl = document.getElementById("cineraIndexControl");
var indexSort = indexControl.querySelector("#cineraIndexSort");
var indexSortChronological = true;

var filterMenu = indexControl.querySelector(".cineraIndexFilter");
if(filterMenu)
{
    var filterContainer = filterMenu.querySelector(".filter_container");
    //menuState.push(linkMenu);

    filterMenu.addEventListener("mouseenter", function(ev) {
        filterContainer.style.display = "block";
    });

    filterMenu.addEventListener("mouseleave", function(ev) {
        filterContainer.style.display = "none";
    });
}

function hideEntriesOfProject(ProjectElement)
{
    if(!ProjectElement.classList.contains("off"))
    {
        ProjectElement.classList.add("off");
    }
    var baseURL = ProjectElement.attributes.getNamedItem("data-baseURL").value;
    var searchLocation = ProjectElement.attributes.getNamedItem("data-searchLocation").value;
    var playerLocation = ProjectElement.attributes.getNamedItem("data-playerLocation").value;
    for(var i = 0; i < projects.length; ++i)
    {
        var ThisProject = projects[i];
        if(baseURL === ThisProject.baseURL && searchLocation === ThisProject.searchLocation && playerLocation === ThisProject.playerLocation)
        {
            ThisProject.filteredOut = true;
            if(ThisProject.entriesContainer != null)
            {
                ThisProject.entriesContainer.style.display = "none";
                disableSprite(ThisProject.entriesContainer.parentElement);
            }
        }
    }
}

function showEntriesOfProject(ProjectElement)
{
    if(ProjectElement.classList.contains("off"))
    {
        ProjectElement.classList.remove("off");
    }
    var baseURL = ProjectElement.attributes.getNamedItem("data-baseURL").value;
    var searchLocation = ProjectElement.attributes.getNamedItem("data-searchLocation").value;
    var playerLocation = ProjectElement.attributes.getNamedItem("data-playerLocation").value;
    for(var i = 0; i < projects.length; ++i)
    {
        var ThisProject = projects[i];
        if(baseURL === ThisProject.baseURL && searchLocation === ThisProject.searchLocation && playerLocation === ThisProject.playerLocation)
        {
            ThisProject.filteredOut = false;
            if(ThisProject.entriesContainer != null)
            {
                ThisProject.entriesContainer.style.display = "flex";
                enableSprite(ThisProject.entriesContainer.parentElement);
            }
        }
    }
}

function hideProjectSearchResults(baseURL, searchLocation, playerLocation)
{
    var cineraResults = document.getElementById("cineraResults");
    if(cineraResults)
    {
        var cineraResultsProjects = cineraResults.querySelectorAll(".projectContainer");
        for(var i = 0; i < cineraResultsProjects.length; ++i)
        {
            var resultBaseURL = cineraResultsProjects[i].attributes.getNamedItem("data-baseURL").value;
            var resultSearchLocation = cineraResultsProjects[i].attributes.getNamedItem("data-searchLocation").value;
            var resultPlayerLocation = cineraResultsProjects[i].attributes.getNamedItem("data-playerLocation").value;
            if(baseURL === resultBaseURL && searchLocation === resultSearchLocation && playerLocation === resultPlayerLocation)
            {
                cineraResultsProjects[i].style.display = "none";
                return;
            }
        }
    }
}

function showProjectSearchResults(baseURL, searchLocation, playerLocation)
{
    var cineraResults = document.getElementById("cineraResults");
    if(cineraResults)
    {
        var cineraResultsProjects = cineraResults.querySelectorAll(".projectContainer");
        for(var i = 0; i < cineraResultsProjects.length; ++i)
        {
            var resultBaseURL = cineraResultsProjects[i].attributes.getNamedItem("data-baseURL").value;
            var resultSearchLocation = cineraResultsProjects[i].attributes.getNamedItem("data-searchLocation").value;
            var resultPlayerLocation = cineraResultsProjects[i].attributes.getNamedItem("data-playerLocation").value;
            if(baseURL === resultBaseURL && searchLocation === resultSearchLocation && playerLocation === resultPlayerLocation)
            {
                cineraResultsProjects[i].style.display = "flex";
                return;
            }
        }
    }
}

function toggleEntriesOfProjectAndChildren(ProjectFilterElement)
{
    var baseURL = ProjectFilterElement.attributes.getNamedItem("data-baseURL").value;
    var searchLocation = ProjectFilterElement.attributes.getNamedItem("data-searchLocation").value;
    var playerLocation = ProjectFilterElement.attributes.getNamedItem("data-playerLocation").value;
    var shouldShow = ProjectFilterElement.classList.contains("off");
    if(shouldShow)
    {
        ProjectFilterElement.classList.remove("off");
        enableSprite(ProjectFilterElement);
    }
    else
    {
        ProjectFilterElement.classList.add("off");
        disableSprite(ProjectFilterElement);
    }

    for(var i = 0; i < projects.length; ++i)
    {
        var ThisProject = projects[i];

        if(baseURL === ThisProject.baseURL && searchLocation === ThisProject.searchLocation && playerLocation === ThisProject.playerLocation)
        {
            if(shouldShow)
            {
                ThisProject.filteredOut = false;
                enableSprite(ThisProject.projectTitleElement.parentElement);
                if(ThisProject.entriesContainer != null)
                {
                    ThisProject.entriesContainer.style.display = "flex";
                }
                showProjectSearchResults(ThisProject.baseURL, ThisProject.searchLocation, ThisProject.playerLocation);
            }
            else
            {
                ThisProject.filteredOut = true;
                disableSprite(ThisProject.projectTitleElement.parentElement);
                if(ThisProject.entriesContainer != null)
                {
                    ThisProject.entriesContainer.style.display = "none";
                }
                hideProjectSearchResults(ThisProject.baseURL, ThisProject.searchLocation, ThisProject.playerLocation);
            }
        }
    }

    var indexChildFilterProjects = ProjectFilterElement.querySelectorAll(".cineraFilterProject");

    for(var j = 0; j < indexChildFilterProjects.length; ++j)
    {
        var ThisElement = indexChildFilterProjects[j];
        var baseURL = ThisElement.attributes.getNamedItem("data-baseURL").value;
        var searchLocation = ThisElement.attributes.getNamedItem("data-searchLocation").value;
        var playerLocation = ThisElement.attributes.getNamedItem("data-playerLocation").value;
        if(shouldShow)
        {
            showEntriesOfProject(ThisElement);
            showProjectSearchResults(baseURL, searchLocation, playerLocation);
        }
        else
        {
            hideEntriesOfProject(ThisElement);
            hideProjectSearchResults(baseURL, searchLocation, playerLocation);
        }
    }
}

var indexFilter = indexControl.querySelector(".cineraIndexFilter");
if(indexFilter)
{
    var indexFilterProjects = indexFilter.querySelectorAll(".cineraFilterProject");
    for(var i = 0; i < indexFilterProjects.length; ++i)
    {
        indexFilterProjects[i].addEventListener("mouseover", function(ev) {
            ev.stopPropagation();
            this.classList.add("focused");
            focusSprite(this);
        });
        indexFilterProjects[i].addEventListener("mouseout", function(ev) {
            ev.stopPropagation();
            this.classList.remove("focused");
            unfocusSprite(this);
        });
        indexFilterProjects[i].addEventListener("click", function(ev) {
            ev.stopPropagation();
            toggleEntriesOfProjectAndChildren(this);
        });
    }
}

var resultsSummary = document.getElementById("cineraResultsSummary");
var resultsContainer = document.getElementById("cineraResults");

var indexContainer = document.getElementById("cineraIndex");

var projectsContainer = indexContainer.querySelectorAll(".cineraIndexProject");

var projectContainerPrototype = document.createElement("DIV");
projectContainerPrototype.classList.add("projectContainer");

var dayContainerPrototype = document.createElement("DIV");
dayContainerPrototype.classList.add("dayContainer");

var dayNamePrototype = document.createElement("SPAN");
dayNamePrototype.classList.add("dayName");
dayContainerPrototype.appendChild(dayNamePrototype);

var markerListPrototype = document.createElement("DIV");
markerListPrototype.classList.add("markerList");
dayContainerPrototype.appendChild(markerListPrototype);

var markerPrototype = document.createElement("A");
markerPrototype.classList.add("marker");
if(resultsContainer.getAttribute("data-single") == 0)
{
    markerPrototype.setAttribute("target", "_blank");
}

function prepareToParseIndexFile(project)
{
    project.xhr.addEventListener("load", function() {
        var contents = project.xhr.response;
        var lines = contents.split("\n");
        var mode = "none";
        var episode = null;
        for (var i = 0; i < lines.length; ++i) {
            var line = lines[i];
            if (line.trim().length == 0) { continue; }
            if (line == "---") {
                if (episode != null && episode.name != null && episode.title != null) {
                    episode.filename = episode.name;
                    episode.day = getEpisodeName(episode.filename + ".html.md");
                    episode.dayContainerPrototype = project.dayContainerPrototype;
                    episode.markerPrototype = markerPrototype;
                    episode.playerURLPrefix = project.playerURLPrefix;
                    project.episodes.push(episode);
                }
                episode = {};
                mode = "none";
            } else if (line.startsWith("name:")) {
                episode.name = line.slice(6);
            } else if (line.startsWith("title:")) {
                episode.title = line.slice(7).trim().slice(1, -1);
            } else if (line.startsWith("markers")) {
                mode = "markers";
                episode.markers = [];
            } else if (mode == "markers") {
                var match = line.match(/"(\d+)": "(.+)"/);
                if (match == null) {
                    console.log(name, line);
                } else {
                    var totalTime = parseInt(line.slice(1));
                    var marker = {
                        totalTime: totalTime,
                        prettyTime: markerTime(totalTime),
                        text: match[2].replace(/\\"/g, "\"")
                    }
                    episode.markers.push(marker);
                }
            }
        }
        document.querySelector(".spinner").classList.remove("show");
        project.parsed = true;
        runSearch(true);
    });
    project.xhr.addEventListener("error", function() {
        console.error("Failed to load content");
    });
}

var projects = [];
function prepareProjects()
{
    for(var i = 0; i < projectsContainer.length; ++i)
    {
        var ID = projectsContainer[i].attributes.getNamedItem("data-project").value;
        var baseURL = projectsContainer[i].attributes.getNamedItem("data-baseURL").value;
        var searchLocation = projectsContainer[i].attributes.getNamedItem("data-searchLocation").value;
        var playerLocation = projectsContainer[i].attributes.getNamedItem("data-playerLocation").value;
        var theme = projectsContainer[i].classList.item(1);

        projects[i] =
            {
                baseURL: baseURL,
                searchLocation: searchLocation,
                playerLocation: playerLocation,
                playerURLPrefix: (baseURL ? baseURL + "/" : "") + (playerLocation ? playerLocation + "/" : ""),
                indexLocation: (baseURL ? baseURL + "/" : "") + (searchLocation ? searchLocation + "/" : "") + ID + ".index",
                projectTitleElement: projectsContainer[i].querySelector(":scope > .cineraProjectTitle"),
                entriesContainer: projectsContainer[i].querySelector(":scope > .cineraIndexEntries"),
                dayContainerPrototype: dayContainerPrototype.cloneNode(true),
                filteredOut: false,
                parsed: false,
                searched: false,
                resultsToRender: [],
                resultsIndex: 0,
                theme: theme,
                episodes: [],
                xhr: new XMLHttpRequest(),
            }

        projects[i].dayContainerPrototype.classList.add(theme);
        projects[i].dayContainerPrototype.children[1].classList.add(theme);

        document.querySelector(".spinner").classList.add("show");
        projects[i].xhr.open("GET", projects[i].indexLocation);
        projects[i].xhr.setRequestHeader("Content-Type", "text/plain");
        projects[i].xhr.send();
        prepareToParseIndexFile(projects[i]);
    }
}
prepareProjects();

indexSort.addEventListener("click", function(ev) {
    if(indexSortChronological)
    {
        this.firstChild.nodeValue = "Sort: New to Old ⏷"
        for(var i = 0; i < projects.length; ++i)
        {
            if(projects[i].entriesContainer)
            {
                projects[i].entriesContainer.style.flexFlow = "column-reverse";
            }
        }
    }
    else
    {
        this.firstChild.nodeValue = "Sort: Old to New ⏶"
        for(var i = 0; i < projects.length; ++i)
        {
            if(projects[i].entriesContainer)
            {
                projects[i].entriesContainer.style.flexFlow = "column";
            }
        }
    }
    indexSortChronological = !indexSortChronological;
    runSearch(true);
});

var lastQuery = null;
var markerList = null;
var projectContainer = null;
var resultsMarkerIndex = -1;
var rendering = false;

var highlightPrototype = document.createElement("B");

function getEpisodeName(filename) {
    var day = filename;
    var dayParts = day.match(/([a-zA-Z_-]+)([0-9]+)?([a-zA-Z]+)?/);
    day = dayParts[1].slice(0, 1).toUpperCase() + dayParts[1].slice(1) + (dayParts[2] ? " " + dayParts[2] : "") + (dayParts[3] ? " " + dayParts[3].toUpperCase() : "");
    return day;
}

function markerTime(totalTime) {
    var markTime = "(";
    var hours = Math.floor(totalTime / 60 / 60);
    var minutes = Math.floor(totalTime / 60) % 60;
    var seconds = totalTime % 60;
    if (hours > 0) {
        markTime += padTimeComponent(hours) + ":";
    }

    markTime += padTimeComponent(minutes) + ":" + padTimeComponent(seconds) + ")";

    return markTime;
}

function padTimeComponent(component) {
    return (component < 10 ? "0" + component : component);
}

function resetProjectsForSearch()
{
    for(var i = 0; i < projects.length; ++i)
    {
        var project = projects[i];
        project.searched = false;
        project.resultsToRender = [];
    }
}

var renderHandle;

function runSearch(refresh) {
    var queryStr = document.getElementById("query").value;
    if (refresh || lastQuery != queryStr) {
        var oldResultsContainer = resultsContainer;
        resultsContainer = oldResultsContainer.cloneNode(false);
        oldResultsContainer.parentNode.insertBefore(resultsContainer, oldResultsContainer);
        oldResultsContainer.remove();
        for(var i = 0; i < projects.length; ++i)
        {
            projects[i].resultsIndex = 0;
        }
        resultsMarkerIndex = -1;
    }
    lastQuery = queryStr;

    resetProjectsForSearch();

    var numEpisodes = 0;
    var numMarkers = 0;
    var totalSeconds = 0;

    // NOTE(matt):  Function defined within runSearch() so that we can modify numEpisodes, numMarkers and totalSeconds
    function runSearchInterior(resultsToRender, query, episode)
    {
        var matches = [];
        for (var k = 0; k < episode.markers.length; ++k) {
            query.lastIndex = 0;
            var result = query.exec(episode.markers[k].text);
            if (result && result[0].length > 0) {
                numMarkers++;
                matches.push(episode.markers[k]);
                if (k < episode.markers.length-1) {
                    totalSeconds += episode.markers[k+1].totalTime - episode.markers[k].totalTime;
                }
            }
        }
        if (matches.length > 0) {
            numEpisodes++;
            resultsToRender.push({
                query: query,
                episode: episode,
                matches: matches
            });
        }
    }

    if (queryStr && queryStr.length > 0) {
        indexContainer.style.display = "none";
        resultsSummary.style.display = "block";
        var shouldRender = false;
        var query = new RegExp(queryStr.replace("(", "\\(").replace(")", "\\)").replace(/\|+/, "\|").replace(/\|$/, "").replace(/(^|[^\\])\\$/, "$1"), "gi");

        // Visible
        for(var i = 0; i < projects.length; ++i)
        {
            var project = projects[i];
            if(project.parsed && !project.filteredOut && project.episodes.length > 0) {
                if(indexSortChronological)
                {
                    for(var j = 0; j < project.episodes.length; ++j) {
                        var episode = project.episodes[j];
                        runSearchInterior(project.resultsToRender, query, episode);
                    }
                }
                else
                {
                    for(var j = project.episodes.length; j > 0; --j) {
                        var episode = project.episodes[j - 1];
                        runSearchInterior(project.resultsToRender, query, episode);
                    }
                }

                shouldRender = true;
                project.searched = true;

            }
        }

        // Invisible
        for(var i = 0; i < projects.length; ++i)
        {
            var project = projects[i];
            if(project.parsed && project.filteredOut && !project.searched && project.episodes.length > 0) {
                if(indexSortChronological)
                {
                    for(var j = 0; j < project.episodes.length; ++j) {
                        var episode = project.episodes[j];
                        runSearchInterior(project.resultsToRender, query, episode);
                    }
                }
                else
                {
                    for(var j = project.episodes.length; j > 0; --j) {
                        var episode = project.episodes[j - 1];
                        runSearchInterior(project.resultsToRender, query, episode);
                    }
                }

                shouldRender = true;
                project.searched = true;

            }
        }

        if(shouldRender)
        {
            if (rendering) {
                clearTimeout(renderHandle);
            }
            renderResults();
        }
    }
    else
    {
        indexContainer.style.display = "block";
        resultsSummary.style.display = "none";
    }

    var totalTime = Math.floor(totalSeconds/60/60) + "h " + Math.floor(totalSeconds/60)%60 + "m " + totalSeconds%60 + "s ";

    resultsSummary.textContent = "Found: " + numEpisodes + " episodes, " + numMarkers + " markers, " + totalTime + "total.";
}

function renderResults() {
    var maxItems = 42;
    var numItems = 0;
    for(var i = 0; i < projects.length; ++i)
    {
        var project = projects[i];
        if (project.resultsIndex < project.resultsToRender.length) {
            rendering = true;
            while (numItems < maxItems && project.resultsIndex < project.resultsToRender.length) {
                var query = project.resultsToRender[project.resultsIndex].query;
                var episode = project.resultsToRender[project.resultsIndex].episode;
                var matches = project.resultsToRender[project.resultsIndex].matches;
                if (resultsMarkerIndex == -1) {
                    if(project.resultsIndex == 0 || project.resultsToRender[project.resultsIndex - 1].episode.playerURLPrefix != episode.playerURLPrefix)
                    {
                        projectContainer = projectContainerPrototype.cloneNode(true);
                        for(var i = 0; i < projects.length; ++i)
                        {
                            if(projects[i].playerURLPrefix === episode.playerURLPrefix)
                            {
                                projectContainer.setAttribute("data-baseURL", projects[i].baseURL);
                                projectContainer.setAttribute("data-searchLocation", projects[i].searchLocation);
                                projectContainer.setAttribute("data-playerLocation", projects[i].playerLocation);
                                if(projects[i].filteredOut)
                                {
                                    projectContainer.style.display = "none";
                                }
                            }
                        }
                        resultsContainer.appendChild(projectContainer);
                    }
                    else
                    {
                        projectContainer = resultsContainer.lastElementChild;
                    }


                    var dayContainer = episode.dayContainerPrototype.cloneNode(true);
                    var dayName = dayContainer.children[0];
                    markerList = dayContainer.children[1];
                    dayName.textContent = episode.day + ": " + episode.title;
                    projectContainer.appendChild(dayContainer);
                    resultsMarkerIndex = 0;
                    numItems++;
                }        

                while (numItems < maxItems && resultsMarkerIndex < matches.length) {
                    var match = matches[resultsMarkerIndex];
                    var marker = episode.markerPrototype.cloneNode(true);
                    marker.setAttribute("href", episode.playerURLPrefix + episode.filename.replace(/"/g, "") + "/#" + match.totalTime);
                        query.lastIndex = 0;
                        var cursor = 0;
                        var text = match.text;
                        var result = null;
                        marker.appendChild(document.createTextNode(match.prettyTime + " "));
                        while (result = query.exec(text)) {
                            if (result.index > cursor) {
                                marker.appendChild(document.createTextNode(text.slice(cursor, result.index)));
                            }
                            var highlightEl = highlightPrototype.cloneNode();
                            highlightEl.textContent = result[0];
                            marker.appendChild(highlightEl);
                            cursor = result.index + result[0].length;
                        }

                    if (cursor < text.length) {
                        marker.appendChild(document.createTextNode(text.slice(cursor, text.length)));
                    }
                    markerList.appendChild(marker);
                    numItems++;
                    resultsMarkerIndex++;
                }

                if (resultsMarkerIndex == matches.length) {
                    resultsMarkerIndex = -1;
                    project.resultsIndex++;
                }
            }
            renderHandle = setTimeout(renderResults, 0);
        } else {
            rendering = false;
        }
    }
}

function IsVisible(el) {
    var xPos = 0;
    var yPos = 0;
    var Height = parseInt(getComputedStyle(el).height);

    while (el) {
        if (el.tagName == "BODY") {
            var xScroll = el.scrollLeft || document.documentElement.scrollLeft;
            var yScroll = el.scrollTop || document.documentElement.scrollTop;

            xPos += (el.offsetLeft - xScroll + el.clientLeft)
            yPos += (el.offsetTop - yScroll + el.clientTop)
        } else {
            xPos += (el.offsetLeft - el.scrollLeft + el.clientLeft);
            yPos += (el.offsetTop - el.scrollTop + el.clientTop);
        }

        el = el.offsetParent;
    }
    return ((xPos > 0 && xPos < window.innerWidth) && (yPos > 0 && yPos + Height < window.innerHeight));
}

var queryEl = document.getElementById("query");
if(document.hasFocus() && IsVisible(queryEl)) { queryEl.focus(); }
queryEl.addEventListener("input", function(ev) {
    history.replaceState(null, null, "#" + encodeURIComponent(queryEl.value));
    runSearch();
});

runSearch();

// Testing
