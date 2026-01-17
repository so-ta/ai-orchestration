// Block Icon composable for Miro-style node design
// Maps icon string names to Lucide Vue components
import type { Component } from 'vue'
import {
  Brain,
  GitBranch,
  Globe,
  Play,
  Code,
  Wrench,
  Clock,
  AlertCircle,
  Workflow,
  Repeat,
  Filter,
  Scissors,
  Layers,
  FileText,
  Shuffle,
  MessageSquare,
  Route,
  Settings,
  Database,
  Sparkles,
  Plug,
  MailIcon,
  TableIcon,
  CheckSquare,
  BookOpen,
  ArrowDownToLine,
  ArrowUpFromLine,
  User,
  CircleDot,
  Zap,
  RefreshCw,
  RotateCcw,
  MessageCircle,
  Hash,
  Search,
  Bot,
  FolderOpen,
  Send,
} from 'lucide-vue-next'

// Map of icon names (strings) to Lucide Vue components
const iconMap: Record<string, Component> = {
  // AI-related icons
  'brain': Brain,
  'sparkles': Sparkles,
  'bot': Bot,
  'message-square': MessageSquare,

  // Flow/Control icons
  'git-branch': GitBranch,
  'shuffle': Shuffle,
  'route': Route,
  'repeat': Repeat,
  'refresh-cw': RefreshCw,
  'rotate-ccw': RotateCcw,
  'workflow': Workflow,

  // Action/Tool icons
  'play': Play,
  'code': Code,
  'wrench': Wrench,
  'zap': Zap,

  // Data icons
  'filter': Filter,
  'scissors': Scissors,
  'layers': Layers,
  'database': Database,
  'arrow-down-to-line': ArrowDownToLine,
  'arrow-up-from-line': ArrowUpFromLine,

  // Status/Utility icons
  'clock': Clock,
  'alert-circle': AlertCircle,
  'settings': Settings,
  'circle-dot': CircleDot,

  // Integration icons
  'globe': Globe,
  'plug': Plug,
  'mail': MailIcon,
  'table': TableIcon,
  'check-square': CheckSquare,
  'book-open': BookOpen,
  'user': User,
  'message-circle': MessageCircle,
  'hash': Hash,
  'search': Search,
  'folder-open': FolderOpen,
  'send': Send,

  // Document icons
  'file-text': FileText,
}

// Default icon mapping by block type (slug)
// Used when block definition doesn't specify an icon
const defaultIconByType: Record<string, string> = {
  // AI blocks
  llm: 'brain',
  router: 'route',

  // Flow/Control blocks
  start: 'play',
  condition: 'git-branch',
  switch: 'shuffle',
  loop: 'repeat',
  map: 'refresh-cw',
  join: 'layers',

  // Data blocks
  filter: 'filter',
  split: 'scissors',
  aggregate: 'layers',
  transform: 'code',

  // Integration blocks
  tool: 'wrench',
  http: 'globe',
  function: 'code',
  subflow: 'workflow',

  // Utility blocks
  wait: 'clock',
  human_in_loop: 'user',
  error: 'alert-circle',
  note: 'file-text',
  log: 'file-text',

  // External integrations
  slack: 'message-circle',
  discord: 'message-circle',
  notion: 'file-text',
  github: 'git-branch',
  google_sheets: 'table',
  linear: 'check-square',
  email: 'mail',
}

/**
 * Get Lucide component for an icon name
 */
export function getIconComponent(iconName: string | undefined): Component {
  if (!iconName) {
    return Code // default fallback
  }
  return iconMap[iconName] || Code
}

/**
 * Get icon name for a block type
 * Uses block's icon field if available, otherwise falls back to default by type
 */
export function getBlockIcon(blockType: string, blockIcon?: string): string {
  if (blockIcon && iconMap[blockIcon]) {
    return blockIcon
  }
  return defaultIconByType[blockType] || 'code'
}

/**
 * Composable to use block icons
 */
export function useBlockIcons() {
  return {
    getIconComponent,
    getBlockIcon,
    iconMap,
    defaultIconByType,
  }
}
