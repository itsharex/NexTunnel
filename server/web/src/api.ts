import type {
  ACLRuleView,
  APIResponse,
  DashboardSnapshot,
  LoginResponse,
  NodeStatus,
  TrafficStats,
  User,
  AlertEvent,
} from './types'

const API_BASE_URL = import.meta.env.VITE_DASHBOARD_API_BASE ?? ''
const AUTH_TOKEN_KEY = 'nextunnel.dashboard.token'

interface RequestOptions extends RequestInit {
  token?: string
}

export const readStoredToken = (): string => localStorage.getItem(AUTH_TOKEN_KEY) ?? ''

export const storeToken = (token: string): void => {
  localStorage.setItem(AUTH_TOKEN_KEY, token)
}

export const clearStoredToken = (): void => {
  localStorage.removeItem(AUTH_TOKEN_KEY)
}

const apiPath = (path: string): string => `${API_BASE_URL}${path}`

const parseJSON = async <T>(response: Response): Promise<APIResponse<T>> => {
  try {
    return (await response.json()) as APIResponse<T>
  } catch {
    return { success: false, error: `HTTP ${response.status}` }
  }
}

const request = async <T>(path: string, options: RequestOptions = {}): Promise<T> => {
  const headers = new Headers(options.headers)
  headers.set('Content-Type', 'application/json')
  if (options.token) {
    headers.set('Authorization', `Bearer ${options.token}`)
  }

  const response = await fetch(apiPath(path), {
    ...options,
    headers,
  })
  const payload = await parseJSON<T>(response)

  if (!response.ok || !payload.success) {
    throw new Error(payload.error ?? `请求失败：HTTP ${response.status}`)
  }
  if (payload.data === undefined) {
    throw new Error('接口未返回有效数据')
  }
  return payload.data
}

export const login = async (username: string, password: string): Promise<LoginResponse> => {
  const data = await request<LoginResponse>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
  storeToken(data.token)
  return data
}

export const fetchHealth = async (): Promise<string> => {
  const data = await request<{ status: string }>('/api/v1/health')
  return data.status
}

export const fetchDashboardSnapshot = async (token: string): Promise<DashboardSnapshot> => {
  const [nodes, stats, aclRules, alerts, users] = await Promise.all([
    request<NodeStatus[]>('/api/v1/nodes', { token }),
    request<TrafficStats[]>('/api/v1/stats', { token }),
    request<ACLRuleView[]>('/api/v1/acl', { token }),
    request<AlertEvent[]>('/api/v1/alerts', { token }),
    request<User[]>('/api/v1/users', { token }),
  ])

  return { nodes, stats, aclRules, alerts, users }
}

export const createACLRule = async (token: string, rule: ACLRuleView): Promise<ACLRuleView> =>
  request<ACLRuleView>('/api/v1/acl', {
    method: 'POST',
    token,
    body: JSON.stringify(rule),
  })

export const deleteACLRule = async (token: string, ruleID: string): Promise<void> => {
  await request<{ deleted: string }>(`/api/v1/acl/${encodeURIComponent(ruleID)}`, {
    method: 'DELETE',
    token,
  })
}

export const deleteNode = async (token: string, nodeID: string): Promise<void> => {
  await request<{ deleted: string }>(`/api/v1/nodes/${encodeURIComponent(nodeID)}`, {
    method: 'DELETE',
    token,
  })
}

export const acknowledgeAlert = async (token: string, alertID: string, ackedBy: string): Promise<void> => {
  await request<{ acked: string }>(`/api/v1/alerts/${encodeURIComponent(alertID)}/ack`, {
    method: 'POST',
    token,
    body: JSON.stringify({ acked_by: ackedBy }),
  })
}
