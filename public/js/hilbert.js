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
