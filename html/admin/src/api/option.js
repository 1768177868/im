import request from '../utils/request'

/**
 * 获取下拉选项数据（统一接口）
 * @param {string} type - 选项类型：role, department, status, method, yes_no, admin
 * @param {Object} extraParams - 额外参数，例如：{ customer_service_only: true }
 * @returns {Promise}
 */
export function getOptions(type, extraParams = {}) {
  return request({
    url: '/options',
    method: 'get',
    params: { type, ...extraParams }
  })
}

