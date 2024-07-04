document.addEventListener("DOMContentLoaded", () => {
  for (const countdown of document.querySelectorAll(".countdown")) {
    const deadline = countdown.getAttribute("data-deadline");
    const deadlineDate = new Date(parseInt(deadline, 10) * 1000);

    function updateCountdown() {
      const remainingMs = deadlineDate.getTime() - new Date().getTime();
      const remainingMinutes = remainingMs / 1000 / 60;
      const remainingHours = remainingMinutes / 60;
      const remainingDays = remainingHours / 24; // no daylight savings transitions during the jam mmkay

      let str = "imminently";
      if (remainingMinutes < 60) {
        str = `in ${Math.ceil(remainingMinutes)} ${
          remainingMinutes === 1 ? "minute" : "minutes"
        }`;
      } else if (remainingHours < 24) {
        str = `in ${Math.ceil(remainingHours)} ${
          remainingHours === 1 ? "hour" : "hours"
        }`;
      } else {
        str = `in ${Math.ceil(remainingDays)} ${
          remainingDays === 1 ? "day" : "days"
        }`;
      }

      countdown.innerText = str;
    }

    updateCountdown();
    setInterval(updateCountdown, 1000 * 60);
  }
});
