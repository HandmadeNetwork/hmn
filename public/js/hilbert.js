function hilbertPath(iterations, step = 10) {
  let l = "A";
  for (let i = 0; i < iterations; i++) {
    l = l.replaceAll(/[AB]/g, c => c === "A" ? "+BF-AFA-FB+" : "-AF+BFB+FA-");
  }

  const dirs = [`h${step}`, `v${step}`, `h-${step}`, `v-${step}`];
  let dir = 0;
  let p = `M${step} ${step}`;
  for (const char of l) {
    if (char === "F") {
      p += dirs[dir];
    } else if (char === "+") {
      dir += 1;
    } else if (char === "-") {
      dir -= 1;
    }
    dir = (dir + 4) % 4;
  }

  return p;
}

function hilbertWidth(iterations, step = 10) {
  return step * (2**iterations - 1 + 2);
}

function createHilbertCurves() {
  const containers = document.querySelectorAll(".hilbert");
  for (const container of containers) {
    const filler = document.createElement("div");
    filler.classList.add("hilbert-filler");
    const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    const path = document.createElementNS("http://www.w3.org/2000/svg", "path");
    const p = hilbertPath(7, 10), width = hilbertWidth(7, 10);
    path.setAttribute("d", p);
    svg.setAttribute("viewBox", `0 0 ${width} ${width}`);
    svg.appendChild(path);
    filler.appendChild(svg);
    container.appendChild(filler);
  }
}

document.addEventListener("DOMContentLoaded", () => createHilbertCurves());
