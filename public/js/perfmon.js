// src/rawdata/js/lib/utils.ts
function assert(cond, msg, soft = false) {
  if (!cond) {
    if (soft) {
      console.error(msg != null ? msg : "Assertion failed");
    } else {
      throw new Error(msg != null ? msg : "Assertion failed");
    }
  }
}

// src/rawdata/js/lib/templates.ts
var templateElementCache = {};
var templatePathCache = {};
function getTemplateEl(id) {
  if (!templateElementCache[id]) {
    const el = document.getElementById(id);
    assert(el, "no element with id ".concat(id));
    assert(el instanceof HTMLTemplateElement);
    templateElementCache[id] = el;
  }
  return templateElementCache[id];
}
function getTemplatePaths(id, rootNode) {
  if (!templatePathCache[id]) {
    let descend2 = function(path, el) {
      for (var i = 0; i < el.children.length; ++i) {
        var child = el.children[i];
        var childPath = path.concat([i]);
        var tmplName = child.getAttribute("data-tmpl");
        if (tmplName) {
          paths.push([tmplName, childPath]);
        }
        if (child.children.length > 0) {
          descend2(childPath, child);
        }
      }
    };
    var descend = descend2;
    var paths = [];
    paths.push(["root", []]);
    descend2([], rootNode);
    templatePathCache[id] = paths;
  }
  return templatePathCache[id];
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
function makeTemplateCloner(id) {
  return function() {
    var templateEl = getTemplateEl(id);
    if (templateEl === null) {
      throw new Error("Couldn't find template with ID '".concat(id, "'"));
    }
    var root = templateEl.content.cloneNode(true);
    var paths = getTemplatePaths(id, root);
    var result = collectElements(paths, root);
    return result;
  };
}

// src/rawdata/js/perfmon.js
function init(perfRecordsJSON) {
  const singleRowTemplate = makeTemplateCloner("single-record");
  const routeTemplate = makeTemplateCloner("route");
  const data = JSON.parse(perfRecordsJSON);
  const recordsContainer = document.querySelector("#records");
  const routesContainer = document.querySelector("#routes");
  let routes = {};
  let recents = [];
  for (let i = data.length - 1; i >= 0; --i) {
    let record = data[i];
    if (recents.length < 100) {
      recents.push(record);
    }
    if (!routes[record.Route]) {
      routes[record.Route] = {
        route: record.Route,
        records: []
      };
    }
    routes[record.Route].records.push(record);
  }
  function showRecords(records) {
    recordsContainer.innerHTML = "";
    for (let i = 0; i < records.length; ++i) {
      let getFlameRow2 = function(idx) {
        while (flameRows.length <= idx) {
          let flameRow = document.createElement("DIV");
          flameRow.classList.add("h1");
          flameRow.classList.add("relative");
          flameRows.push(flameRow);
          flameGraph.appendChild(flameRow);
        }
        return flameRows[idx];
      }, placeFlameItems2 = function(parent, depth, maxDuration) {
        let rowEl = getFlameRow2(depth);
        for (let childIdx = 0; childIdx < parent.Children.length; ++childIdx) {
          let child = parent.Children[childIdx];
          let item = document.createElement("DIV");
          let catCSS = child.Category.toLowerCase();
          item.classList.add("absolute", "h1", "overflow-hidden", "f7", "nowrap");
          item.style.color = "var(--".concat(catCSS, "-text)");
          item.style.backgroundColor = "var(--".concat(catCSS, "-bg, var(--c1))");
          item.textContent = "[".concat(child.Category, "] ").concat(child.Description, " | ").concat(child.Duration / 1e3, "ms");
          item.title = item.textContent;
          item.style.width = child.Duration / maxDuration * 100 + "%";
          item.style.left = child.Offset / maxDuration * 100 + "%";
          rowEl.appendChild(item);
          for (const checkpoint of child.Checkpoints) {
            const el = document.createElement("div");
            el.title = "[".concat(checkpoint.Category, "] ").concat(checkpoint.Description);
            el.classList.add("absolute", "h1", "f7", "nowrap");
            el.style.backgroundColor = "#00000066";
            el.style.width = "2px";
            el.style.left = checkpoint.Offset / maxDuration * 100 + "%";
            rowEl.appendChild(el);
          }
          if (child.Children && child.Children.length > 0) {
            placeFlameItems2(child, depth + 1, maxDuration);
          }
        }
      };
      var getFlameRow = getFlameRow2, placeFlameItems = placeFlameItems2;
      let record = records[i];
      let row = singleRowTemplate();
      row.route.textContent = record.Route;
      row.duration.textContent = record.Duration / 1e3 + "ms";
      row.path.textContent = record.Path;
      let flameGraph = row.flamegraph;
      let flameRows = [];
      if (record.Breakdown.Children) {
        placeFlameItems2(record.Breakdown, 0, record.Breakdown.Duration);
      }
      recordsContainer.appendChild(row.row);
    }
  }
  for (const key in routes) {
    routes[key].records.sort(function(a, b) {
      return b.Duration - a.Duration;
    });
  }
  let routesList = Object.values(routes);
  routesList.sort(function(a, b) {
    const medianA = a.records[Math.floor(a.records.length / 2)].Duration;
    const medianB = b.records[Math.floor(b.records.length / 2)].Duration;
    return medianB - medianA;
  });
  let activeRoute = "";
  for (let i = 0; i < routesList.length; ++i) {
    let r = routesList[i];
    let routeEl = routeTemplate();
    routeEl.name.textContent = r.route;
    routeEl.hits.textContent = r.records.length;
    routeEl.duration.textContent = r.records[Math.floor(r.records.length / 2)].Duration / 1e3 + "ms";
    routeEl.route.dataset.route = r.route;
    routesContainer.appendChild(routeEl.route);
    routesList[i].el = routeEl;
    routeEl.route.addEventListener("click", function(ev) {
      let el = ev.target.closest(".route");
      if (el.dataset.route == activeRoute) {
        el.classList.remove("bg4");
        activeRoute = "";
        showRecords(recents);
      } else {
        for (let i2 = 0; i2 < routesList.length; ++i2) {
          routesList[i2].el.route.classList.remove("bg4");
        }
        activeRoute = el.dataset.route;
        el.classList.add("bg4");
        showRecords(routes[activeRoute].records);
      }
    });
  }
  showRecords(recents);
}
export {
  init
};
