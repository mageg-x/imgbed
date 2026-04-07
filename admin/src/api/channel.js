import request from './request'

export const channelApi = {
  list() {
    return request.get('/admin/channels')
  },

  get(id) {
    return request.get(`/admin/channels/${id}`)
  },

  create(data) {
    return request.post('/admin/channels', data)
  },

  update(id, data) {
    return request.put(`/admin/channels/${id}`, data)
  },

  delete(id) {
    return request.delete(`/admin/channels/${id}`)
  },

  enable(id) {
    return request.put(`/admin/channels/${id}/enable`, { enabled: true })
  },

  disable(id) {
    return request.put(`/admin/channels/${id}/enable`, { enabled: false })
  },

  test(id) {
    return request.post(`/admin/channels/${id}/test`)
  },

  getQuota(id) {
    return request.get(`/admin/channels/${id}/quota`)
  },

  getStats(id) {
    return request.get(`/admin/channels/${id}/stats`)
  }
}
