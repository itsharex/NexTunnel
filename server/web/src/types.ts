export interface APIResponse<T> {
  success: boolean
  data?: T
  error?: string
}

export interface User {
  id: string
  username: string
  role: string
  email: string
}

export interface LoginResponse {
  token: string
  expires_at: string
  user: User
}

export interface NodeStatus {
  node_id: string
  region: string
  nat_type: string
  online: boolean
  connected_at: string
  last_seen: string
  rx_bytes: number
  tx_bytes: number
}

export interface TrafficStats {
  node_id?: string
  rx_bytes: number
  tx_bytes: number
  rx_bandwidth_bps: number
  tx_bandwidth_bps: number
  connections: number
  timestamp: string
}

export interface ACLRuleView {
  id: string
  source: string
  target: string
  action: string
  protocol: string
  priority: number
  enabled: boolean
  created_at: string
}

export interface AlertEvent {
  id: string
  rule_id?: string
  rule_name?: string
  level: string
  message: string
  node_id?: string
  value?: number
  threshold?: number
  created_at: string
  acked: boolean
  acked_by?: string
  acked_at?: string
}

export interface DashboardSnapshot {
  nodes: NodeStatus[]
  stats: TrafficStats[]
  aclRules: ACLRuleView[]
  alerts: AlertEvent[]
  users: User[]
}

export interface ACLFormState {
  source: string
  target: string
  action: string
  protocol: string
  priority: number
  enabled: boolean
}
