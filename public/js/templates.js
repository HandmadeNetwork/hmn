var domLookupCache = {};
var templatePathCache = {};

function getTemplateEl(id) {
    if (!domLookupCache[id]) {
        domLookupCache[id] = document.getElementById(id);
    }
    return domLookupCache[id];
}

function collectElements(paths, rootElement) {
    var result = {};
    for (var i = 0; i < paths.length; ++i) {
        var path = paths[i];
        var current = rootElement;
        for (var j = 0; j < path[1].length; ++j) {
            current = current.children[path[1][j]];
        }
        result[path[0]] = current;
    }
    return result;
}

function getTemplatePaths(id, rootElement) {
    if (!templatePathCache[id]) {
        var paths = [];
        paths.push(["root", []]);

        function descend(path, el) {
            for (var i = 0; i < el.children.length; ++i) {
                var child = el.children[i];
                var childPath = path.concat([i]);
                var tmplName = child.getAttribute("data-tmpl");
                if (tmplName) {
                    paths.push([tmplName, childPath]);
                }
                if (child.children.length > 0) {
                    descend(childPath, child);
                }
            }
        }

        descend([], rootElement);
        templatePathCache[id] = paths;
    }
    return templatePathCache[id];
}

function makeTemplateCloner(id) {
    return function() {
        var templateEl = getTemplateEl(id);
        if (templateEl === null) {
            throw new Error(`Couldn\'t find template with ID '${id}'`);
        }

        var root = templateEl.content.cloneNode(true);
        var paths = getTemplatePaths(id, root);
        var result = collectElements(paths, root);
        return result;
    };
}

function emptyElement(el) {
    var newEl = el.cloneNode(false);
    el.parentElement.insertBefore(newEl, el);
    el.parentElement.removeChild(el);
    return newEl;
}
