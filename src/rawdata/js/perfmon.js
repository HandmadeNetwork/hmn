import { makeTemplateCloner } from "./lib/templates";

export function init(perfRecordsJSON) {
  const singleRowTemplate = makeTemplateCloner("single-record");
  const routeTemplate = makeTemplateCloner("route");
  const data = JSON.parse(perfRecordsJSON);
  const recordsContainer = document.querySelector("#records");
  const routesContainer = document.querySelector("#routes");

  let routes = {}
  let recents = [];

  for (let i = data.length - 1; i >= 0; --i) {
    let record = data[i];

    if (recents.length < 100) {
      recents.push(record);
    }

    if (!routes[record.Route]) {
      routes[record.Route] = {
        route: record.Route,
        records: [],
      };
    }

    routes[record.Route].records.push(record);
  }

  function showRecords(records) {
    recordsContainer.innerHTML = "";
    for (let i = 0; i < records.length; ++i) {
      let record = records[i];
      let row = singleRowTemplate();
      row.route.textContent = record.Route;
      row.duration.textContent = record.Duration / 1000 + "ms";
      row.path.textContent = record.Path;
      let flameGraph = row.flamegraph;

      let flameRows = [];
      function getFlameRow(idx) {
        while (flameRows.length <= idx) {
          let flameRow = document.createElement("DIV");
          flameRow.classList.add("h1");
          flameRow.classList.add("relative");
          flameRows.push(flameRow);
          flameGraph.appendChild(flameRow);
        }

        return flameRows[idx];
      }

      function placeFlameItems(parent, depth, maxDuration) {
        let rowEl = getFlameRow(depth);
        for (let childIdx = 0; childIdx < parent.Children.length; ++childIdx) {
          let child = parent.Children[childIdx];
          let item = document.createElement("DIV");
          let catCSS = child.Category.toLowerCase();
          item.classList.add("absolute", "h1", "overflow-hidden", "f7", "nowrap");
          item.style.color = `var(--${catCSS}-text)`;
          item.style.backgroundColor = `var(--${catCSS}-bg, var(--c1))`;
          item.textContent = `[${child.Category}] ${child.Description} | ${child.Duration / 1000}ms`;
          item.title = item.textContent;
          item.style.width = ((child.Duration / maxDuration) * 100) + "%";
          item.style.left = (((child.Offset) / maxDuration) * 100) + "%";
          rowEl.appendChild(item);

          for (const checkpoint of child.Checkpoints) {
            const el = document.createElement("div");
            el.title = `[${checkpoint.Category}] ${checkpoint.Description}`;
            el.classList.add("absolute", "h1", "f7", "nowrap");
            el.style.backgroundColor = `#00000066`;
            el.style.width = "2px";
            el.style.left = (((checkpoint.Offset) / maxDuration) * 100) + "%";
            rowEl.appendChild(el);
          }

          if (child.Children && child.Children.length > 0) {
            placeFlameItems(child, depth + 1, maxDuration);
          }
        }
      }

      if (record.Breakdown.Children) {
        placeFlameItems(record.Breakdown, 0, record.Breakdown.Duration);
      }

      recordsContainer.appendChild(row.row);
    }
  }

  for (const key in routes) {
    routes[key].records.sort(function (a, b) {
      return b.Duration - a.Duration;
    });
  }

  let routesList = Object.values(routes);
  routesList.sort(function (a, b) {
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
    routeEl.duration.textContent = (r.records[Math.floor(r.records.length / 2)].Duration / 1000) + "ms";
    routeEl.route.dataset.route = r.route;
    routesContainer.appendChild(routeEl.route);
    routesList[i].el = routeEl;

    routeEl.route.addEventListener("click", function (ev) {
      let el = ev.target.closest(".route");
      if (el.dataset.route == activeRoute) {
        el.classList.remove("bg4");
        activeRoute = "";
        showRecords(recents);
      } else {
        for (let i = 0; i < routesList.length; ++i) {
          routesList[i].el.route.classList.remove("bg4");
        }
        activeRoute = el.dataset.route;
        el.classList.add("bg4");
        showRecords(routes[activeRoute].records);
      }
    });
  }

  showRecords(recents);
}
