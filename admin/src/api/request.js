import axios from "axios";
import { useAuthStore } from "@/stores/auth";
import { ElMessage } from "element-plus";

const request = axios.create({
  baseURL: "/api/v1",
  timeout: 30000,
  withCredentials: true,
});

request.interceptors.response.use(
  (response) => {
    return response.data;
  },
  (error) => {
    const { response } = error;
    if (response) {
      const { status, data } = response;
      if (status === 401) {
        const authStore = useAuthStore();
        authStore.logout();
        // 只有不在登录页才跳转
        if (!window.location.pathname.startsWith("/admin/login")) {
          window.location.href = "/admin/login";
        }
      } else if (status === 403) {
        const authStore = useAuthStore();
        authStore.logout();
        window.location.href = "/admin/login";
        return;
      } else if (status === 404) {
        ElMessage.error("请求的资源不存在");
      } else {
        ElMessage.error(data?.message || "请求失败");
      }
      return Promise.reject(data);
    }
    ElMessage.error("网络错误，请检查网络连接");
    return Promise.reject(error);
  },
);

export default request;
