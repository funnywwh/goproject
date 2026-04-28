import request from '../utils/request'

export type BugStatus = 'active' | 'resolved' | 'closed'

export interface Bug {
  id: number
  title: string
  description?: string
  status: BugStatus
  priority: 'low' | 'medium' | 'high' | 'urgent'
  severity: 'low' | 'medium' | 'high' | 'critical'
  confirmed?: boolean  // 是否确认
  project_id: number
  project?: any
  creator_id: number
  creator?: any
  assignees?: any[]
  requirement_id?: number
  requirement?: any
  module_id?: number
  module?: any
  estimated_hours?: number
  actual_hours?: number
  solution?: string
  solution_note?: string
  resolved_version_id?: number
  resolved_version?: any
  versions?: any[]  // 所属版本列表（多对多关系）
  attachments?: any[]  // 附件列表（多对多关系）
  created_at?: string
  updated_at?: string
}

export interface BugListResponse {
  list: Bug[]
  total: number
  page: number
  page_size: number
}

export interface CreateBugRequest {
  title: string
  description?: string
  status?: BugStatus
  priority?: 'low' | 'medium' | 'high' | 'urgent'
  severity?: 'low' | 'medium' | 'high' | 'critical'
  project_id: number
  requirement_id?: number
  module_id?: number
  assignee_ids?: number[]
  version_ids: number[]  // 所属版本ID列表（必填，至少一个）
  estimated_hours?: number
  actual_hours?: number
  work_date?: string
}

export interface UpdateBugStatusRequest {
  status: BugStatus
  solution?: string
  solution_note?: string
  estimated_hours?: number
  actual_hours?: number
  work_date?: string
  resolved_version_id?: number
  version_number?: string
  create_version?: boolean
}

export interface AssignBugRequest {
  assignee_ids: number[]
  status?: BugStatus
  comment?: string
}

// Bug相关API
export const getBugs = async (params?: {
  keyword?: string
  project_id?: number
  status?: string | BugStatus[]
  priority?: string
  severity?: string
  requirement_id?: number
  module_id?: number
  creator_id?: number
  assignee_id?: number
  version_id?: number  // 版本ID筛选
  page?: number
  size?: number
}): Promise<BugListResponse> => {
  const requestParams = params ? { ...params } : undefined
  if (requestParams && Array.isArray(requestParams.status)) {
    requestParams.status = requestParams.status.join(',')
  }
  return request.get('/bugs', { params: requestParams })
}

export const getBug = async (id: number): Promise<Bug> => {
  return request.get(`/bugs/${id}`)
}

export const createBug = async (data: CreateBugRequest): Promise<Bug> => {
  return request.post('/bugs', data)
}

export interface UpdateBugRequest extends Partial<CreateBugRequest> {
  version_ids?: number[]  // 所属版本ID列表（可选，更新时提供）
  attachment_ids?: number[]  // 附件ID列表（可选，更新时提供）
}

export const updateBug = async (id: number, data: UpdateBugRequest): Promise<Bug> => {
  // 确保 requirement_id 和 module_id 字段始终被包含，即使值为 0
  // 使用 Object.assign 确保字段被显式设置
  const requestData: any = Object.assign({}, data)
  
  // 始终包含 requirement_id 字段，如果未定义或为 null，设置为 0
  // 注意：即使值为 0，也要确保字段存在
  if (data.requirement_id === undefined || data.requirement_id === null) {
    requestData.requirement_id = 0
  } else {
    requestData.requirement_id = data.requirement_id
  }
  
  // 始终包含 module_id 字段，如果未定义或为 null，设置为 0
  if (data.module_id === undefined || data.module_id === null) {
    requestData.module_id = 0
  } else {
    requestData.module_id = data.module_id
  }
  
  // 确保 attachment_ids 字段始终被包含，即使是空数组也要发送
  // 注意：必须显式设置，不能依赖 Object.assign，因为 undefined 值可能被忽略
  if (data.attachment_ids !== undefined && data.attachment_ids !== null) {
    requestData.attachment_ids = Array.isArray(data.attachment_ids) ? data.attachment_ids : []
  } else {
    // 如果没有提供 attachment_ids，设置为空数组，确保后端能正确处理
    requestData.attachment_ids = []
  }
  
  return request.put(`/bugs/${id}`, requestData)
}

export const deleteBug = async (id: number): Promise<void> => {
  return request.delete(`/bugs/${id}`)
}

export const updateBugStatus = async (id: number, data: UpdateBugStatusRequest): Promise<Bug> => {
  return request.patch(`/bugs/${id}/status`, data)
}

export const assignBug = async (id: number, data: AssignBugRequest): Promise<Bug> => {
  return request.post(`/bugs/${id}/assign`, data)
}

export const confirmBug = async (id: number): Promise<Bug> => {
  return request.post(`/bugs/${id}/confirm`)
}

export interface BugStatistics {
  total: number
  active: number  // 激活状态
  resolved: number
  closed: number
  low_priority: number
  medium_priority: number
  high_priority: number
  urgent_priority: number
  low_severity: number
  medium_severity: number
  high_severity: number
  critical_severity: number
}

export const getBugStatistics = async (params?: {
  keyword?: string
  project_id?: number
  requirement_id?: number
  creator_id?: number
}): Promise<BugStatistics> => {
  return request.get('/bugs/statistics', { params })
}

// 历史记录相关接口
export interface History {
  id: number
  action_id: number
  field: string
  old: string
  old_value: string
  new: string
  new_value: string
  diff?: string
  created_at?: string
}

export interface Action {
  id: number
  object_type: string
  object_id: number
  project_id: number
  actor_id: number
  actor?: {
    id: number
    username: string
    nickname?: string
  }
  action: 'created' | 'edited' | 'assigned' | 'resolved' | 'closed' | 'confirmed' | 'commented'
  date: string
  comment?: string
  extra?: string
  histories?: History[]
  created_at?: string
}

export interface BugHistoryResponse {
  list: Action[]
}

export interface AddBugHistoryNoteRequest {
  comment: string
}

export const getBugHistory = async (id: number): Promise<BugHistoryResponse> => {
  return request.get(`/bugs/${id}/history`)
}

export const addBugHistoryNote = async (id: number, data: AddBugHistoryNoteRequest): Promise<{ message: string }> => {
  return request.post(`/bugs/${id}/history/note`, data)
}

// 列设置相关接口
export interface ColumnSetting {
  key: string
  visible: boolean
  order: number
  width?: number
}

export const getBugColumnSettings = async (): Promise<ColumnSetting[]> => {
  return request.get('/bugs/column-settings')
}

export const saveBugColumnSettings = async (settings: ColumnSetting[]): Promise<void> => {
  return request.post('/bugs/column-settings', settings)
}
