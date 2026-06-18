export type ScenicVisualType = 'landmark' | 'experience' | 'culture'

export type StructuredScenicSpot = {
  id: string
  name: string
  area: '灵山胜境' | '拈花湾'
  category: string
  visualType: ScenicVisualType
  description: string
  lng: number
  lat: number
  rating: number
  price: number
  imageUrl: string
  parameters: string[]
  culture: string
  highlights: string[]
  openInfo: string
  showTimes: string[]
  routeTags: Array<'history' | 'nature' | 'family'>
  thumbnail: string
  geofenceEnabled: boolean
  geofenceRadiusM: number
  geofenceIntroText: string
  geofenceCooldownMinutes: number
  signalBlindSpot?: boolean
}

export type ScenicRoutePlan = {
  id: 'history' | 'nature' | 'family'
  name: string
  duration: string
  summary: string
  spotIds: string[]
  nodeHighlights: Record<string, string>
}

export type ServiceReminder = {
  spotId: string
  title: string
  startTime: string
  advanceMinutes: number
  message: string
  priority: 'high' | 'medium' | 'low'
}

export type GuideInsight = {
  title: string
  image: string
  tags: string[]
  points: string[]
}

export const SCENIC_SPOTS: StructuredScenicSpot[] = [
  {
    id: 'LS-001',
    name: '灵山大佛',
    area: '灵山胜境',
    category: '地标建筑',
    visualType: 'landmark',
    description: '灵山胜境核心地标，以大佛主体、登云道和礼佛广场构成强识别游览节点。',
    lng: 120.4204,
    lat: 31.5681,
    rating: 4.9,
    price: 0,
    imageUrl: '',
    thumbnail: '佛',
    parameters: ['总高101.5m', '用铜725吨', '216级台阶'],
    culture: '216级台阶对应108烦恼与108愿望，登临过程强调礼佛与自省。',
    highlights: ['仰观大佛全景', '登云道礼佛动线', '适合数字人讲解高度与台阶寓意'],
    openInfo: '随景区开放，建议上午或傍晚避开强逆光时段。',
    showTimes: [],
    routeTags: ['history', 'family'],
    geofenceEnabled: true,
    geofenceRadiusM: 120,
    geofenceIntroText: '灵山大佛总高101.5米，用铜725吨，216级台阶寓意化解108烦恼并承载108愿望。',
    geofenceCooldownMinutes: 180,
  },
  {
    id: 'LS-006',
    name: '九龙灌浴',
    area: '灵山胜境',
    category: '演艺体验',
    visualType: 'experience',
    description: '动态音乐喷泉与佛教典故结合的核心体验点，是亲子游客高频停留节点。',
    lng: 120.4215,
    lat: 31.5667,
    rating: 4.8,
    price: 0,
    imageUrl: '',
    thumbnail: '浴',
    parameters: ['总高27.2m', '动态喷泉表演', '表演后可接祈福圣水'],
    culture: '以“九龙灌浴”典故呈现佛诞场景，适合用短讲解降低文化理解门槛。',
    highlights: ['平日多场演出', '喷泉与音乐同步', '亲子路线核心点位'],
    openInfo: '平日演出10:00/11:30/13:30/15:00，节假日以现场公告为准。',
    showTimes: ['10:00', '11:30', '13:30', '15:00'],
    routeTags: ['family', 'nature'],
    geofenceEnabled: true,
    geofenceRadiusM: 100,
    geofenceIntroText: '九龙灌浴总高27.2米，平日10:00、11:30、13:30、15:00演出，表演后可接祈福圣水。',
    geofenceCooldownMinutes: 120,
  },
  {
    id: 'LS-003',
    name: '梵宫',
    area: '灵山胜境',
    category: '地标建筑',
    visualType: 'landmark',
    description: '集建筑艺术、文化展陈与圣坛演出于一体的室内核心建筑。',
    lng: 120.423,
    lat: 31.5674,
    rating: 4.8,
    price: 0,
    imageUrl: '',
    thumbnail: '宫',
    parameters: ['大型文化建筑', '圣坛演出空间', '室内参观动线'],
    culture: '以佛教艺术、雕塑、穹顶与舞台演出呈现东方文化审美。',
    highlights: ['适合避雨避暑', '亲子互动演出', '历史文化讲解密度高'],
    openInfo: '《灵山吉祥颂》演出建议提前30分钟排队入场。',
    showTimes: ['灵山吉祥颂演出前30分钟排队'],
    routeTags: ['history', 'family'],
    geofenceEnabled: true,
    geofenceRadiusM: 110,
    geofenceIntroText: '梵宫融合建筑艺术和圣坛演出，观看《灵山吉祥颂》建议提前排队。',
    geofenceCooldownMinutes: 180,
    signalBlindSpot: true,
  },
  {
    id: 'LS-004',
    name: '五明桥',
    area: '灵山胜境',
    category: '文化休憩',
    visualType: 'culture',
    description: '连接礼佛动线的文化桥梁节点，适合短停讲解和拍照。',
    lng: 120.4195,
    lat: 31.5672,
    rating: 4.6,
    price: 0,
    imageUrl: '',
    thumbnail: '桥',
    parameters: ['汉白玉雕刻', '5座并列', '桥面文化纹饰'],
    culture: '五明桥代表佛教五种智慧，过桥寓意开启智慧。',
    highlights: ['适合AR到点提示', '文化寓意清晰', '连接大佛与核心游线'],
    openInfo: '全天随步行游线开放，雨天注意桥面湿滑。',
    showTimes: [],
    routeTags: ['history', 'nature'],
    geofenceEnabled: true,
    geofenceRadiusM: 80,
    geofenceIntroText: '五明桥代表佛教五种智慧，过桥寓意开启智慧；桥体为汉白玉雕刻，5座并列。',
    geofenceCooldownMinutes: 180,
  },
  {
    id: 'NH-005',
    name: '五灯湖',
    area: '拈花湾',
    category: '演艺体验',
    visualType: 'experience',
    description: '拈花湾夜游核心水域，承载《禅行》灯光秀与夜间游览体验。',
    lng: 120.4079,
    lat: 31.4954,
    rating: 4.7,
    price: 0,
    imageUrl: '',
    thumbnail: '湖',
    parameters: ['夜间水景灯光', '湖岸观演空间', '夜游客流集中'],
    culture: '用灯光、水景与行进式观演表达禅意夜游氛围。',
    highlights: ['19:00/20:00《禅行》灯光秀', '适合夜游提醒', '适合增设临时休息区'],
    openInfo: '夜间19:00/20:00《禅行》灯光秀，建议提前抵达湖岸。',
    showTimes: ['19:00《禅行》', '20:00《禅行》'],
    routeTags: ['nature', 'family'],
    geofenceEnabled: true,
    geofenceRadiusM: 120,
    geofenceIntroText: '五灯湖夜间19:00和20:00有《禅行》灯光秀，建议提前抵达湖岸观演。',
    geofenceCooldownMinutes: 180,
  },
  {
    id: 'LS-008',
    name: '祥符禅寺',
    area: '灵山胜境',
    category: '文化休憩',
    visualType: 'culture',
    description: '传统禅寺文化点位，适合安静参访、祈福与历史文化讲解。',
    lng: 120.4183,
    lat: 31.569,
    rating: 4.6,
    price: 0,
    imageUrl: '',
    thumbnail: '寺',
    parameters: ['传统寺院空间', '祈福参访节点', '低噪游览区'],
    culture: '承载寺院礼仪、祈福习俗与佛教文化背景。',
    highlights: ['适合历史文化路线', '适合低强度步行', '可承接素斋咨询'],
    openInfo: '随景区开放，参访时保持安静。',
    showTimes: [],
    routeTags: ['history'],
    geofenceEnabled: true,
    geofenceRadiusM: 90,
    geofenceIntroText: '祥符禅寺适合安静参访，可了解寺院礼仪、祈福习俗与佛教文化背景。',
    geofenceCooldownMinutes: 180,
  },
  {
    id: 'NH-003',
    name: '百子戏弥勒',
    area: '拈花湾',
    category: '文化休憩',
    visualType: 'culture',
    description: '适合亲子互动和文化拍照的轻量游览点。',
    lng: 120.4069,
    lat: 31.4963,
    rating: 4.7,
    price: 0,
    imageUrl: '',
    thumbnail: '童',
    parameters: ['亲子拍照点', '互动雕塑群', '低强度停留'],
    culture: '用童趣形象降低文化理解门槛，适合家庭游客停留。',
    highlights: ['亲子路线重点', '拍照停留', '与梵宫互动演出形成组合'],
    openInfo: '全天开放，建议白天游览。',
    showTimes: [],
    routeTags: ['family'],
    geofenceEnabled: true,
    geofenceRadiusM: 90,
    geofenceIntroText: '百子戏弥勒适合亲子互动和拍照，可作为家庭游客的轻松停留点。',
    geofenceCooldownMinutes: 180,
  },
  {
    id: 'NH-007',
    name: '无尽意斋',
    area: '拈花湾',
    category: '文化休憩',
    visualType: 'culture',
    description: '素斋与休憩咨询高频点，适合承接餐饮推荐和体力恢复。',
    lng: 120.4091,
    lat: 31.4945,
    rating: 4.5,
    price: 0,
    imageUrl: '',
    thumbnail: '斋',
    parameters: ['素斋推荐', '休憩补给', '夜游前后停留'],
    culture: '结合素食文化与慢游体验，适合数字人进行餐饮建议。',
    highlights: ['素斋咨询高频', '休息补给', '夜游路线补充点'],
    openInfo: '餐饮开放以现场营业时间为准。',
    showTimes: [],
    routeTags: ['nature', 'family'],
    geofenceEnabled: false,
    geofenceRadiusM: 100,
    geofenceIntroText: '',
    geofenceCooldownMinutes: 1440,
  },
]

export const SCENIC_ROUTES: ScenicRoutePlan[] = [
  {
    id: 'history',
    name: '历史文化路线',
    duration: '约3小时',
    summary: '突出大佛、梵宫、五明桥、祥符禅寺的文化内涵和建筑参数。',
    spotIds: ['LS-004', 'LS-001', 'LS-003', 'LS-008'],
    nodeHighlights: {
      'LS-004': '五明桥：五种智慧与汉白玉桥体参数。',
      'LS-001': '灵山大佛：101.5m高度、725吨用铜、216级台阶寓意。',
      'LS-003': '梵宫：建筑艺术与《灵山吉祥颂》观演提醒。',
      'LS-008': '祥符禅寺：寺院礼仪与祈福文化。',
    },
  },
  {
    id: 'nature',
    name: '自然风光路线',
    duration: '约2.5小时',
    summary: '降低高密度讲解，突出桥、水景、夜游和休憩节奏。',
    spotIds: ['LS-004', 'LS-006', 'NH-005', 'NH-007'],
    nodeHighlights: {
      'LS-004': '五明桥：过桥寓意开启智慧，适合拍照停留。',
      'LS-006': '九龙灌浴：喷泉表演与祈福圣水体验。',
      'NH-005': '五灯湖：夜间《禅行》灯光秀与湖岸动线。',
      'NH-007': '无尽意斋：素斋补给和夜游休憩。',
    },
  },
  {
    id: 'family',
    name: '亲子路线',
    duration: '约2小时',
    summary: '优先体验九龙灌浴、百子戏弥勒、梵宫等亲子友好点位。',
    spotIds: ['LS-006', 'NH-003', 'LS-003', 'LS-001'],
    nodeHighlights: {
      'LS-006': '九龙灌浴：固定演出时段清晰，孩子容易理解。',
      'NH-003': '百子戏弥勒：互动拍照和轻松停留。',
      'LS-003': '梵宫：室内参观与互动演出，适合避雨避暑。',
      'LS-001': '灵山大佛：用高度、台阶数字做短讲解。',
    },
  },
]

export const SERVICE_REMINDERS: ServiceReminder[] = [
  {
    spotId: 'LS-006',
    title: '九龙灌浴演出提醒',
    startTime: '13:30',
    advanceMinutes: 10,
    message: '距离九龙灌浴演出约10分钟，建议前往观赏，表演后可接祈福圣水。',
    priority: 'high',
  },
  {
    spotId: 'LS-003',
    title: '灵山吉祥颂排队提醒',
    startTime: '14:00',
    advanceMinutes: 30,
    message: '梵宫圣坛需提前排队，建议现在前往。',
    priority: 'medium',
  },
  {
    spotId: 'NH-005',
    title: '五灯湖夜游提醒',
    startTime: '19:00',
    advanceMinutes: 20,
    message: '五灯湖19:00/20:00有《禅行》灯光秀，湖岸客流较大，建议提前抵达。',
    priority: 'high',
  },
]

export function findStructuredSpot(idOrName: string) {
  return SCENIC_SPOTS.find(spot => spot.id === idOrName || spot.name === idOrName)
}

export function buildGuideInsight(text: string): GuideInsight | null {
  const content = text.toLowerCase()
  const hit = SCENIC_SPOTS.find(spot => {
    const terms = [spot.id.toLowerCase(), spot.name.toLowerCase(), ...spot.parameters.map(item => item.toLowerCase())]
    return terms.some(term => term && content.includes(term))
  })
  if (!hit) return null
  return {
    title: hit.name,
    image: hit.thumbnail,
    tags: [hit.area, hit.category, hit.openInfo].filter(Boolean).slice(0, 3),
    points: [...hit.parameters, ...hit.highlights].slice(0, 5),
  }
}
