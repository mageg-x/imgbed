import request from './request'

export function createBackup() {
  return request({
    url: '/admin/backup/create',
    method: 'post'
  })
}

export function listBackups() {
  return request({
    url: '/admin/backup/list',
    method: 'get'
  })
}

export function deleteBackup(backupPath) {
  return request({
    url: '/admin/backup',
    method: 'delete',
    data: {
      backup_path: backupPath
    }
  })
}

export function restoreBackup(backupPath) {
  return request({
    url: '/admin/backup/restore',
    method: 'post',
    data: {
      backup_path: backupPath
    }
  })
}
