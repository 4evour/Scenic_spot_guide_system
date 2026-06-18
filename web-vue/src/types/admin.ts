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

export type VisitorInsightAnalysis = {
  id: number
  user_id: number
  session_id: string
  summary: string
  satisfaction_score: number
  negative_reasons: string
  attention_points: string
  raw_result: string
  status: string
  created_at: string
  updated_at: string
}

export type AvatarConfig = {
  default_avatar_id: string
  allow_avatar_switch: boolean
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
  period?: '7d' | '30d'
  attention_analysis: Array<{ label: string; value: number }>
  emotion_distribution: Array<{ label: string; icon: string; count: number; percent: number }>
  emotion_trend: Array<{ date: string; positive_rate: number; negative_rate: number; total: number }>
  negative_reasons?: Array<{ label: string; value: number }>
  audience_profiles?: Array<{ label: string; percent: number; route: string; satisfaction: number }>
  route_satisfaction?: Array<{ label: string; clickRate: number; satisfaction: number }>
  word_cloud?: Array<{ label: string; value: number }>
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

export type VisitorQuery = {
  [key: string]: unknown
  id: number
  query: string
  response: string
  spot_id: number
  is_answered: boolean
  created_at: string
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
  default_avatar_id: 'mao_pro',
  allow_avatar_switch: true,
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
  negative_reasons: [],
  audience_profiles: [],
  route_satisfaction: [],
  word_cloud: [],
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
