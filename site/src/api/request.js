import axios from "axios";
import { ElMessage } from "element-plus";
import i18n from "@/i18n";

const request = axios.create({
  baseURL: "/api/v1",
  timeout: 60000,
  withCredentials: true,
});

// 解析详细错误消息，返回用户友好的翻译
function parseDetailedError(msg) {
  const { t } = i18n.global;
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
        document.cookie = "imgbed_token=;expires=Thu, 01 Jan 1970 00:00:00 GMT;path=/";
      } else if (status === 403) {
        document.cookie = "imgbed_token=;expires=Thu, 01 Jan 1970 00:00:00 GMT;path=/";
        window.location.href = '/login'
        return
      } else {
        const detailedError = parseDetailedError(data?.message)
        ElMessage.error(detailedError || data?.message || i18n.global.t('error.requestFailed'));
      }
      return Promise.reject(data);
    }
    ElMessage.error(i18n.global.t('error.network'));
    return Promise.reject(error);
  },
);

export default request;
