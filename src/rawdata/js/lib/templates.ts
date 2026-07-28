import { assert } from "./utils";

export type TemplatePath = [name: string, path: number[]];
export type ClonedTemplate = { [name: string]: Node };

var templateElementCache: { [id: string]: HTMLTemplateElement } = {};
var templatePathCache: { [id: string]: TemplatePath[] } = {};

function getTemplateEl(id: string): HTMLTemplateElement {
    if (!templateElementCache[id]) {
        const el = document.getElementById(id);
        assert(el, `no element with id ${id}`);
        assert(el instanceof HTMLTemplateElement);
        templateElementCache[id] = el as HTMLTemplateElement;
    }
    return templateElementCache[id];
}

function getTemplatePaths(id: string, rootNode: ParentNode) {
    if (!templatePathCache[id]) {
        var paths: TemplatePath[] = [];
        paths.push(["root", []]);

        function descend(path: number[], el: ParentNode) {
            for (var i = 0; i < el.children.length; ++i) {
                var child = el.children[i];
                var childPath: number[] = path.concat([i]);
                var tmplName = child.getAttribute("data-tmpl");
                if (tmplName) {
                    paths.push([tmplName, childPath]);
                }
                if (child.children.length > 0) {
                    descend(childPath, child);
                }
            }
        }

        descend([], rootNode);
        templatePathCache[id] = paths;
    }
    return templatePathCache[id];
}

function collectElements(paths: TemplatePath[], rootElement: ParentNode) {
    var result: { [name: string]: Node } = {};
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

export function makeTemplateCloner<T extends ClonedTemplate>(id: string): () => T & { root: DocumentFragment } {
    return function () {
        var templateEl = getTemplateEl(id);
        if (templateEl === null) {
            throw new Error(`Couldn\'t find template with ID '${id}'`);
        }

        var root = templateEl.content.cloneNode(true) as DocumentFragment;
        var paths = getTemplatePaths(id, root);
        var result = collectElements(paths, root);
        return result as T & { root: DocumentFragment };
    };
}

export function emptyElement(el: Node) {
    var newEl = el.cloneNode(false);
    assert(el.parentElement);
    el.parentElement.insertBefore(newEl, el);
    el.parentElement.removeChild(el);
    return newEl;
}
