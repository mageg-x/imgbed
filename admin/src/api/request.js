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

  // 解析详细错误消息
  const msg = data?.message || '';
  return parseDetailedError(msg, t) || t('error.requestFailed');
}

// 解析详细错误消息，返回用户友好的翻译
function parseDetailedError(msg, t) {
  if (!msg) return null

  // 上传相关错误
  if (msg.includes('upload') || msg.includes('Upload')) {
    if (msg.includes('retry') || msg.includes('Retry')) {
      return t('error.upload.retryExhausted')
    }
    if (msg.includes('Method Not Allowed') || msg.includes('405')) {
      return t('error.upload.methodNotAllowed')
    }
    if (msg.includes('AccessDenied') || msg.includes('access denied') || msg.includes('403')) {
      return t('error.upload.accessDenied')
    }
    if (msg.includes('No such bucket') || msg.includes('BucketNotFound') || msg.includes('404')) {
      return t('error.upload.channelError')
    }
    if (msg.includes('QuotaFull') || msg.includes('quota') || msg.includes('storage full')) {
      return t('error.upload.quotaFull')
    }
    if (msg.includes('RateLimit') || msg.includes('rate limit') || msg.includes('429')) {
      return t('error.upload.rateLimit')
    }
    if (msg.includes('timeout') || msg.includes('Timeout') || msg.includes('ETIMEDOUT')) {
      return t('error.upload.timeout')
    }
    if (msg.includes('network') || msg.includes('Network') || msg.includes('ECONNREFUSED') || msg.includes('ENOTFOUND')) {
      return t('error.upload.networkError')
    }
    if (msg.includes('FileTooLarge') || msg.includes('EntityTooLarge') || msg.includes('file too large')) {
      return t('error.upload.fileTooLarge')
    }
    if (msg.includes('InvalidContentType') || msg.includes('UnsupportedMediaType')) {
      return t('error.upload.invalidFileType')
    }
    if (msg.includes('500') || msg.includes('InternalError')) {
      return t('error.upload.serverError')
    }
    return t('error.upload.failed')
  }

  // 删除相关错误
  if (msg.includes('delete') || msg.includes('Delete')) {
    if (msg.includes('not found') || msg.includes('Not Found') || msg.includes('404')) {
      return t('error.delete.notFound')
    }
    if (msg.includes('AccessDenied') || msg.includes('access denied') || msg.includes('403')) {
      return t('error.delete.accessDenied')
    }
    if (msg.includes('500') || msg.includes('InternalError')) {
      return t('error.delete.serverError')
    }
    return t('error.delete.failed')
  }

  // 下载相关错误
  if (msg.includes('download') || msg.includes('Download')) {
    if (msg.includes('not found') || msg.includes('Not Found') || msg.includes('404')) {
      return t('error.download.notFound')
    }
    if (msg.includes('AccessDenied') || msg.includes('access denied') || msg.includes('403')) {
      return t('error.download.accessDenied')
    }
    if (msg.includes('500') || msg.includes('InternalError')) {
      return t('error.download.serverError')
    }
    return t('error.download.failed')
  }

  return null
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
