import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { authApi } from "@/api/auth";

export const useAuthStore = defineStore("auth", () => {
  const token = ref("");
  const user = ref(null);

  const isAuthenticated = computed(() => !!token.value);

  async function login(password) {
    const res = await authApi.login(password);
    if (res.code === 0) {
      token.value = "authenticated";
      return { success: true };
    }
    return { success: false, message: res.message };
  }

  async function adminLogin(credentials) {
    const res = await authApi.adminLogin(
      credentials.username,
      credentials.password,
    );
    if (res.code === 0) {
      token.value = "authenticated";
      return { success: true };
    }
    return { success: false, message: res.message };
  }

  function logout() {
    token.value = "";
    user.value = null;
    document.cookie = "imgbed_token=;expires=Thu, 01 Jan 1970 00:00:00 GMT;path=/";
  }

  async function checkSession() {
    try {
      const res = await authApi.check();
      if (res.code === 0) {
        token.value = "authenticated";
        user.value = res.data;
        return true;
      }
      logout();
      return false;
    } catch {
      logout();
      return false;
    }
  }

  return { token, user, isAuthenticated, login, adminLogin, logout, checkSession };
});
