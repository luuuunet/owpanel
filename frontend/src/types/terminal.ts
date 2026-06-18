import { uniqueId } from '@/utils/uniqueId'

export interface TerminalTarget {
  id: string
  label: string
  host: string
  port: number
  user: string
  node_id?: number
  asset_id?: number
  account_id?: number
  has_password?: boolean
  is_local?: boolean
  permission?: string
}

export interface TerminalSession {
  id: string
  title: string
  selectedTarget: string
  host: string
  port: number
  user: string
  password: string
  authMethod: 'password' | 'key'
  keyId: number | null
  privateKeyPaste: string
  keyPassphrase: string
  assetId: number | null
  accountId: number | null
  connected: boolean
  connecting: boolean
  /** Background output since last focus */
  unread?: boolean
  connectedAt?: number
}

export function createSession(index: number, target?: TerminalTarget): TerminalSession {
  const host = target?.host || '127.0.0.1'
  const port = target?.port || 22
  const user = target?.user || 'root'
  return {
    id: uniqueId(),
    title: target?.label || `SSH ${index}`,
    selectedTarget: target?.id || 'local',
    host,
    port,
    user,
    password: '',
    authMethod: 'password',
    keyId: null,
    privateKeyPaste: '',
    keyPassphrase: '',
    assetId: target?.asset_id || null,
    accountId: target?.account_id || null,
    connected: false,
    connecting: false,
  }
}
