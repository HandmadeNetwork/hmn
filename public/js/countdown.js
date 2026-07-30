// src/rawdata/js/lib/utils.ts
function assert(cond, msg, soft = false) {
  if (!cond) {
    if (soft) {
      console.error(msg ?? "Assertion failed");
    } else {
      throw new Error(msg ?? "Assertion failed");
    }
  }
}
function must(val, msg) {
  assert(val, msg);
  return val;
}

// src/rawdata/js/countdown.ts
document.addEventListener("DOMContentLoaded", () => {
  for (const countdown of document.querySelectorAll(".countdown")) {
    let updateCountdown2 = function() {
      const remainingMs = deadlineDate.getTime() - (/* @__PURE__ */ new Date()).getTime();
      const remainingMinutes = remainingMs / 1e3 / 60;
      const remainingHours = remainingMinutes / 60;
      const remainingDays = remainingHours / 24;
      let str = "imminently";
      if (remainingMinutes < 60) {
        str = `in ${Math.ceil(remainingMinutes)} ${remainingMinutes === 1 ? "minute" : "minutes"}`;
      } else if (remainingHours < 24) {
        str = `in ${Math.ceil(remainingHours)} ${remainingHours === 1 ? "hour" : "hours"}`;
      } else {
        str = `in ${Math.ceil(remainingDays)} ${remainingDays === 1 ? "day" : "days"}`;
      }
      countdown.innerText = str;
    };
    var updateCountdown = updateCountdown2;
    const deadline = must(countdown.getAttribute("data-deadline"));
    const deadlineDate = new Date(parseInt(deadline, 10) * 1e3);
    updateCountdown2();
    setInterval(updateCountdown2, 1e3 * 60);
  }
});
