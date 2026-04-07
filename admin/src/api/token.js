import request from './request'

export const tokenApi = {
  list() {
    return request.get('/admin/tokens')
  },

  get(token) {
    return request.get(`/admin/tokens/${token}`)
  },

  create(data) {
    return request.post('/admin/tokens', data)
  },

  delete(token) {
    return request.delete(`/admin/tokens/${token}`)
  },

  toggle(token, enabled) {
    return request.put(`/admin/tokens/${token}/enable`, { enabled })
  }
}
