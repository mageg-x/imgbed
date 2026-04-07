import request from './request'

export const fileApi = {
  list(params) {
    return request.get('/admin/files', { params })
  },

  getInfo(id) {
    return request.get(`/admin/files/${id}/info`)
  },

  delete(id) {
    return request.delete(`/admin/files/${id}`)
  },

  batchDelete(ids) {
    return request.post('/admin/files/batch-delete', { ids })
  },

  rename(id, name) {
    return request.put(`/admin/files/${id}/rename`, { name })
  },

  move(id, directory) {
    return request.put(`/admin/files/${id}/move`, { directory })
  },

  updateTags(id, tags) {
    return request.put(`/admin/files/${id}/tags`, { tags })
  },

  block(id) {
    return request.put(`/admin/files/${id}/block`)
  },

  unblock(id) {
    return request.delete(`/admin/files/${id}/block`)
  },

  whitelist(id) {
    return request.put(`/admin/files/${id}/whitelist`)
  },

  removeFromWhitelist(id) {
    return request.delete(`/admin/files/${id}/whitelist`)
  }
}
