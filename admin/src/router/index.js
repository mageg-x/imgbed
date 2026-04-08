import { createRouter, createWebHistory } from "vue-router";
import { useAuthStore } from "@/stores/auth";

const routes = [
  {
    path: "/login",
    name: "Login",
    component: () => import("@/views/Login.vue"),
    meta: { title: 'routeTitle.login', requiresAuth: false },
  },
  {
    path: "/",
    redirect: "/home",
  },
  {
    path: "/",
    component: () => import("@/layouts/MainLayout.vue"),
    meta: { requiresAuth: true },
    children: [
      {
        path: "files",
        name: "Files",
        component: () => import("@/views/Files.vue"),
        meta: { title: 'routeTitle.files' },
      },
      {
        path: "channels",
        name: "Channels",
        component: () => import("@/views/Channels.vue"),
        meta: { title: 'routeTitle.channels' },
      },
      {
        path: "settings",
        name: "Settings",
        component: () => import("@/views/Settings.vue"),
        meta: { title: 'routeTitle.settings' },
      },
      {
        path: "tokens",
        name: "Tokens",
        component: () => import("@/views/Tokens.vue"),
        meta: { title: 'routeTitle.tokens' },
      },
      {
        path: "integration",
        name: "Integration",
        component: () => import("@/views/Integration.vue"),
        meta: { title: 'routeTitle.integration' },
      },
      {
        path: "home",
        name: "Dashboard",
        component: () => import("@/views/Dashboard.vue"),
        meta: { title: 'routeTitle.dashboard' },
      },
    ],
  },
];

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
});

router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore();

  if (!authStore.initialized) {
    await authStore.checkSession();
  }

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next({ name: "Login" });
  } else if (to.name === "Login" && authStore.isAuthenticated && authStore.user?.role === 'admin') {
    next({ name: "Dashboard" });
  } else {
    next();
  }
});

export default router;
