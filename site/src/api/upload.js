import request from "./request";

export const uploadApi = {
  upload(file, options = {}) {
    const formData = new FormData();
    formData.append("file", file);

    if (options.directory) {
      formData.append("directory", options.directory);
    }
    if (options.tags?.length > 0) {
      formData.append("tags", options.tags.join(","));
    }
    if (options.channel) {
      formData.append("channel", options.channel);
    }

    return request.post("/upload", formData, {
      headers: { "Content-Type": "multipart/form-data" },
      onUploadProgress: (progressEvent) => {
        if (options.onProgress && progressEvent.total) {
          const progress = Math.round(
            (progressEvent.loaded * 100) / progressEvent.total,
          );
          options.onProgress(progress);
        }
      },
    });
  },

  initChunk(
    filename,
    fileSize,
    chunkSize = 5 * 1024 * 1024,
    directory = "",
    tags = [],
  ) {
    return request.post("/upload/chunk/init", {
      filename,
      fileSize,
      chunkSize,
      directory,
      tags,
    });
  },

  uploadChunk(uploadId, chunkIndex, chunk) {
    const formData = new FormData();
    formData.append("chunk", chunk);
    return request.post(`/upload/chunk/${uploadId}/${chunkIndex}`, formData, {
      headers: { "Content-Type": "multipart/form-data" },
    });
  },

  mergeChunk(uploadId) {
    return request.post(`/upload/chunk/${uploadId}/merge`);
  },

  cancelChunk(uploadId) {
    return request.delete(`/upload/chunk/${uploadId}`);
  },
};
