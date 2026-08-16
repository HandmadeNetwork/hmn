// src/rawdata/js/hilbert.ts
function hilbertPath(iterations, step) {
  let l = "A";
  for (let i = 0; i < iterations; i++) {
    l = l.replaceAll(/[AB]/g, (c) => c === "A" ? "+BF-AFA-FB+" : "-AF+BFB+FA-");
  }
  return l;
}
function hilbertWidth(iterations, step) {
  return step * (2 ** iterations - 1);
}
function createHilbertCurves() {
  const containers = document.querySelectorAll(".hilbert");
  for (const container of containers) {
    let redrawCanvas2 = function() {
      const pxWidth = canvas.clientWidth * window.devicePixelRatio;
      const pxHeight = canvas.clientHeight * window.devicePixelRatio;
      canvas.width = pxWidth;
      canvas.height = pxHeight;
      const defaultSize = Math.max(window.innerWidth * 1.2, 1920);
      const angle = parseInt(container.dataset.angle ?? "8", 10);
      const size = parseInt(container.dataset.size ?? `${defaultSize}`, 10);
      const iterations = 7;
      const step = 10;
      const p = hilbertPath(iterations, step);
      const width = hilbertWidth(iterations, step);
      ctx.resetTransform();
      ctx.translate(pxWidth / 2, pxHeight / 2);
      ctx.rotate(-angle);
      ctx.scale(size / width * window.devicePixelRatio, size / width * window.devicePixelRatio);
      let x = -width / 2, y = -width / 2;
      ctx.moveTo(x, y);
      const dirs = [
        [step, 0],
        [0, step],
        [-step, 0],
        [0, -step]
      ];
      let dir = 0;
      for (const char of p) {
        if (char === "F") {
          const [dx, dy] = dirs[dir];
          x += dx;
          y += dy;
          ctx.lineTo(x, y);
        } else if (char === "+") {
          dir += 1;
        } else if (char === "-") {
          dir -= 1;
        }
        dir = (dir + 4) % 4;
      }
      ctx.strokeStyle = getComputedStyle(canvas).color;
      ctx.lineWidth = 2;
      ctx.stroke();
    };
    var redrawCanvas = redrawCanvas2;
    const canvas = document.createElement("canvas");
    container.appendChild(canvas);
    const ctx = canvas.getContext("2d");
    window.addEventListener("resize", redrawCanvas2);
    window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", redrawCanvas2);
    redrawCanvas2();
  }
}
document.addEventListener("DOMContentLoaded", createHilbertCurves);
