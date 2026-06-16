export type KnowledgeItem = {
  id: string
  title: string
  source: string
  content: string
  category: string
  knowledge_category: string
  spot_id: number
  spot_category: string
  metadata: string
  updated: string
}

export type KnowledgeCandidate = {
  id: number
  analysis_id: number
  session_id: string
  title: string
  content: string
  source: string
  knowledge_category: string
  spot_id: number
  spot_category: string
  status: string
  reject_reason: string
  created_at: string
}

export type AvatarConfig = {
  name: string
  appearance: string
  costume: string
  style: string
  color: string
  culture_theme: string
  voice_type: string
  voice_tone: string
  speed: number
  volume: number
  greeting: string
  default_emotion: string
  emotion_level: number
}

export type VisitorReport = {
  attention_analysis: Array<{ label: string; value: number }>
  emotion_distribution: Array<{ label: string; icon: string; count: number; percent: number }>
  emotion_trend: Array<{ date: string; positive_rate: number; negative_rate: number; total: number }>
  suggestions: Array<{ content: string }>
  peak_hours: Array<{ hour: string; count: number }>
  summary: {
    total_interactions: number
    satisfaction_rate: number
    negative_rate: number
    top_concern: string
    peak_hour: string
  }
}

export type SystemSettings = {
  scenic_name: string
  scenic_desc: string
  service_hotline: string
  enable_login: boolean
  enable_voice: boolean
  enable_filter: boolean
  backup_frequency: string
  data_retention: string
}

export const defaultAvatarConfig: AvatarConfig = {
  name: '小灵',
  appearance: '亲和型国风讲解员',
  costume: '古典汉服',
  style: '古典汉服',
  color: '#D4AF37',
  culture_theme: '',
  voice_type: '温柔自然女声',
  voice_tone: '温暖、端庄、亲切',
  speed: 0.8,
  volume: 80,
  greeting: '',
  default_emotion: 'joy',
  emotion_level: 3,
}

export const defaultVisitorReport: VisitorReport = {
  attention_analysis: [],
  emotion_distribution: [],
  emotion_trend: [],
  suggestions: [],
  peak_hours: [],
  summary: {
    total_interactions: 0,
    satisfaction_rate: 0,
    negative_rate: 0,
    top_concern: '暂无数据',
    peak_hour: '暂无数据',
  },
}

export const defaultSettings: SystemSettings = {
  scenic_name: '',
  scenic_desc: '',
  service_hotline: '',
  enable_login: true,
  enable_voice: true,
  enable_filter: true,
  backup_frequency: '每日',
  data_retention: '30',
}
