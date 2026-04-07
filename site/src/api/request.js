import axios from "axios";
import { ElMessage } from "element-plus";

const request = axios.create({
  baseURL: "/api/v1",
  timeout: 60000,
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
        document.cookie = "imgbed_token=;expires=Thu, 01 Jan 1970 00:00:00 GMT;path=/";
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
