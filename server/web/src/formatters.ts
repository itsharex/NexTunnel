const BYTE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB'] as const
const BANDWIDTH_UNITS = ['bps', 'Kbps', 'Mbps', 'Gbps', 'Tbps'] as const
const DEFAULT_LOCALE = 'zh-CN'
const MINUTE_MS = 60_000

export const formatBytes = (value: number): string => {
  if (!Number.isFinite(value) || value <= 0) {
    return '0 B'
  }
  let normalizedValue = value
  let unitIndex = 0
  while (normalizedValue >= 1024 && unitIndex < BYTE_UNITS.length - 1) {
    normalizedValue /= 1024
    unitIndex += 1
  }
  return `${normalizedValue.toFixed(normalizedValue >= 10 || unitIndex === 0 ? 0 : 1)} ${BYTE_UNITS[unitIndex]}`
}

export const formatBandwidth = (value: number): string => {
  if (!Number.isFinite(value) || value <= 0) {
    return '0 bps'
  }
  let normalizedValue = value
  let unitIndex = 0
  while (normalizedValue >= 1000 && unitIndex < BANDWIDTH_UNITS.length - 1) {
    normalizedValue /= 1000
    unitIndex += 1
  }
  return `${normalizedValue.toFixed(normalizedValue >= 10 || unitIndex === 0 ? 0 : 1)} ${BANDWIDTH_UNITS[unitIndex]}`
}

export const formatNumber = (value: number): string =>
  new Intl.NumberFormat(DEFAULT_LOCALE, { maximumFractionDigits: 0 }).format(value)

export const formatDateTime = (value: string): string => {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return '未知'
  }
  return new Intl.DateTimeFormat(DEFAULT_LOCALE, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(parsed)
}

export const formatRelativeTime = (value: string): string => {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return '未知'
  }

  const diffMinutes = Math.max(0, Math.floor((Date.now() - parsed.getTime()) / MINUTE_MS))
  if (diffMinutes < 1) {
    return '刚刚'
  }
  if (diffMinutes < 60) {
    return `${diffMinutes} 分钟前`
  }

  const diffHours = Math.floor(diffMinutes / 60)
  if (diffHours < 24) {
    return `${diffHours} 小时前`
  }

  return formatDateTime(value)
}

export const statusLabel = (online: boolean): string => (online ? '在线' : '离线')
