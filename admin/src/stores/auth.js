import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { authApi } from "@/api/auth";

export const useAuthStore = defineStore("auth", () => {
  const token = ref("");
  const user = ref(null);
  const initialized = ref(false);

  const isAuthenticated = computed(() => !!token.value);

  async function login(username, password) {
    try {
      const res = await authApi.login(username, password);
      if (res.code === 0) {
        token.value = "authenticated";
        initialized.value = true;
        return { success: true };
      }
      return { success: false, message: res.message };
    } catch (error) {
      return { success: false, message: error.message };
    }
  }

  async function logout() {
    try {
      await authApi.logout()
    } catch {}
    token.value = "";
    user.value = null;
    initialized.value = true;
    document.cookie = "imgbed_token=;expires=Thu, 01 Jan 1970 00:00:00 GMT;path=/";
  }

  async function checkSession() {
    try {
      const res = await authApi.session();
      if (res.code === 0) {
        token.value = "authenticated";
        user.value = res.data;
        initialized.value = true;
        return true;
      }
      logout();
      return false;
    } catch {
      logout();
      return false;
    }
  }

  return {
    token,
    user,
    initialized,
    isAuthenticated,
    login,
    logout,
    checkSession,
  };
});
