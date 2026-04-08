import axios from "axios";
import { useAuthStore } from "@/stores/auth";
import { ElMessage } from "element-plus";
import i18n from "@/i18n";

const request = axios.create({
  baseURL: "/api/v1",
  timeout: 30000,
  withCredentials: true,
});

// 获取翻译后的错误消息
function getTranslatedErrorMessage(data) {
  const { t } = i18n.global;

  // 如果后端返回了 code，尝试翻译
  if (data?.code !== undefined && data?.code !== 0) {
    const codeKey = `error.code.${data.code}`;
    const translated = t(codeKey);
    // 如果翻译结果和 key 相同，说明没有找到翻译，使用后端消息
    if (translated !== codeKey) {
      return translated;
    }
  }

  // 否则使用后端返回的 message
  return data?.message || t('error.requestFailed');
}

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
        ElMessage.error(i18n.global.t('error.requestResourceNotFound'));
      } else {
        ElMessage.error(getTranslatedErrorMessage(data));
      }
      return Promise.reject(data);
    }
    ElMessage.error(i18n.global.t('error.network'));
    return Promise.reject(error);
  },
);

export default request;
