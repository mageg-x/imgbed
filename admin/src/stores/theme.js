import { defineStore } from "pinia";
import { ref } from "vue";

export const useThemeStore = defineStore("theme", () => {
  const isDark = ref(false);

  function init() {
    const savedTheme = localStorage.getItem("theme");
    if (savedTheme === "dark") {
      isDark.value = true;
      document.documentElement.classList.remove("light");
    } else {
      isDark.value = false;
      document.documentElement.classList.add("light");
    }
  }

  function toggle() {
    isDark.value = !isDark.value;
    if (isDark.value) {
      document.documentElement.classList.remove("light");
    } else {
      document.documentElement.classList.add("light");
    }
    localStorage.setItem("theme", isDark.value ? "dark" : "light");
  }

  return { isDark, init, toggle };
});
