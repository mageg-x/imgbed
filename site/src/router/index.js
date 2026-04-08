import { createRouter, createWebHistory } from "vue-router";
import { useAuthStore } from "@/stores/auth";

const routes = [
  {
    path: "/",
    name: "Home",
    component: () => import("@/views/Home.vue"),
    meta: { title: 'routeTitle.uploadFile' },
  },
  {
    path: "/login",
    name: "Login",
    component: () => import("@/views/Login.vue"),
    meta: { title: 'routeTitle.login' },
  },
  {
    path: "/gallery",
    name: "Gallery",
    component: () => import("@/views/Gallery.vue"),
    meta: { title: 'routeTitle.gallery', requiresAuth: true },
  },
  {
    path: "/browse",
    name: "Browse",
    component: () => import("@/views/Browse.vue"),
    meta: { title: 'routeTitle.browse', requiresAuth: true },
  },
];

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) return savedPosition;
    return { top: 0 };
  },
});

router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore();

  // If not initialized, check session
  if (!authStore.isAuthenticated) {
    await authStore.checkSession()
  }

  // All non-login pages check for admin user
  if (to.name !== "Login" && authStore.user?.role === 'admin') {
    next({ name: "Login" });
    return
  }

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next({ name: "Login", query: { redirect: to.fullPath } });
  } else {
    next();
  }
});

export default router;
