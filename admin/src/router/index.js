import { createRouter, createWebHistory } from "vue-router";
import { useAuthStore } from "@/stores/auth";

const routes = [
  {
    path: "/login",
    name: "Login",
    component: () => import("@/views/Login.vue"),
    meta: { title: "登录", requiresAuth: false },
  },
  {
    path: "/admin",
    redirect: "/admin/"
  },
  {
    path: "/",
    component: () => import("@/layouts/MainLayout.vue"),
    meta: { requiresAuth: true },
    children: [
      {
        path: "",
        name: "Dashboard",
        component: () => import("@/views/Dashboard.vue"),
        meta: { title: "仪表盘" },
      },
      {
        path: "files",
        name: "Files",
        component: () => import("@/views/Files.vue"),
        meta: { title: "文件管理" },
      },
      {
        path: "channels",
        name: "Channels",
        component: () => import("@/views/Channels.vue"),
        meta: { title: "渠道管理" },
      },
      {
        path: "settings",
        name: "Settings",
        component: () => import("@/views/Settings.vue"),
        meta: { title: "系统设置" },
      },
      {
        path: "tokens",
        name: "Tokens",
        component: () => import("@/views/Tokens.vue"),
        meta: { title: "API Token" },
      },
      {
        path: "integration",
        name: "Integration",
        component: () => import("@/views/Integration.vue"),
        meta: { title: "集成示例" },
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
  } else if (to.name === "Login" && authStore.isAuthenticated) {
    next({ name: "Dashboard" });
  } else {
    next();
  }
});

export default router;
