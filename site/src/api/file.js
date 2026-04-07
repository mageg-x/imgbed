import request from "./request";

export const fileApi = {
  list(params) {
    return request.get("/files", { params });
  },
  getInfo(id) {
    return request.get(`/file/${id}/info`);
  },
  getUrl(id) {
    return request.get(`/file/${id}/url`);
  },
  deleteFile(id) {
    return request.delete(`/file/${id}`);
  },
  deleteMultiple(ids) {
    return request.delete("/files", { data: { ids } });
  },
  checkExists(checksum) {
    return request.get(`/file/check/${checksum}`);
  },
};
