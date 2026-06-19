export type ScenicVisualType = 'landmark' | 'experience' | 'culture'
type LocaleCode = 'zh-CN' | 'en-US'

export type StructuredScenicSpot = {
  id: string
  name: string
  area: string
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
  translations?: Partial<Record<LocaleCode, Partial<StructuredScenicSpot>>>
}

export type ScenicRoutePlan = {
  id: 'history' | 'nature' | 'family'
  name: string
  duration: string
  summary: string
  spotIds: string[]
  nodeHighlights: Record<string, string>
  translations?: Partial<Record<LocaleCode, Partial<ScenicRoutePlan>>>
}

export type ServiceReminder = {
  spotId: string
  title: string
  startTime: string
  advanceMinutes: number
  message: string
  priority: 'high' | 'medium' | 'low'
  translations?: Partial<Record<LocaleCode, Partial<ServiceReminder>>>
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
    translations: {
      'en-US': {
        name: 'Lingshan Grand Buddha',
        area: 'Lingshan Scenic Area',
        category: 'Landmark',
        description: 'The core landmark of Lingshan, formed by the Buddha statue, Dengyun Path, and worship square.',
        parameters: ['101.5m tall', '725 tons of copper', '216 steps'],
        culture: 'The 216 steps echo 108 worries and 108 wishes, turning the climb into a moment of reflection.',
        highlights: ['Full Buddha panorama', 'Dengyun worship route', 'Height and step symbolism for guided narration'],
        openInfo: 'Open with the scenic area. Visit in the morning or late afternoon to avoid strong backlight.',
        geofenceIntroText: 'The Lingshan Grand Buddha is 101.5 meters tall, uses 725 tons of copper, and its 216 steps symbolize resolving 108 worries while carrying 108 wishes.',
      },
    },
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
    translations: {
      'en-US': {
        name: 'Nine Dragons Bathing',
        area: 'Lingshan Scenic Area',
        category: 'Performance',
        description: 'A signature experience combining a musical fountain with a Buddhist story, often favored by families.',
        thumbnail: '9D',
        parameters: ['27.2m tall', 'Dynamic fountain show', 'Blessing water after the show'],
        culture: 'The show presents the Buddhist birth story through movement and music, making the cultural meaning easy to follow.',
        highlights: ['Multiple daily shows', 'Synchronized fountain and music', 'Key stop on family routes'],
        openInfo: 'Shows usually run at 10:00, 11:30, 13:30, and 15:00. Holiday schedules follow on-site notices.',
        geofenceIntroText: 'Nine Dragons Bathing is 27.2 meters tall, with shows at 10:00, 11:30, 13:30, and 15:00 on regular days. Visitors can receive blessing water after the show.',
      },
    },
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
    translations: {
      'en-US': {
        name: 'Brahma Palace',
        area: 'Lingshan Scenic Area',
        category: 'Landmark',
        description: 'An indoor landmark combining architecture, cultural exhibitions, and the Sacred Altar performance.',
        thumbnail: 'Pal',
        parameters: ['Large cultural building', 'Sacred Altar performance space', 'Indoor visit route'],
        culture: 'Buddhist art, sculpture, domes, and stage performance present an Eastern cultural aesthetic.',
        highlights: ['Good for rain or heat', 'Family-friendly performance', 'Dense historical and cultural narration'],
        openInfo: 'Arrive about 30 minutes early for the Auspicious Ode of Lingshan performance.',
        showTimes: ['Queue 30 minutes before Auspicious Ode of Lingshan'],
        geofenceIntroText: 'Brahma Palace combines architectural art and Sacred Altar performance. Queue early for Auspicious Ode of Lingshan.',
      },
    },
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
    translations: {
      'en-US': {
        name: 'Wuming Bridge',
        area: 'Lingshan Scenic Area',
        category: 'Culture & Rest',
        description: 'A cultural bridge on the worship route, suited for short stops, narration, and photos.',
        thumbnail: 'Br',
        parameters: ['White marble carving', 'Five parallel bridges', 'Cultural bridge patterns'],
        culture: 'Wuming Bridge represents five forms of Buddhist wisdom, and crossing it symbolizes opening wisdom.',
        highlights: ['Good for AR arrival prompts', 'Clear cultural symbolism', 'Connects the Buddha and core route'],
        openInfo: 'Open along the walking route. Watch for slippery surfaces on rainy days.',
        geofenceIntroText: 'Wuming Bridge represents five forms of Buddhist wisdom. Crossing it symbolizes opening wisdom; the bridge is carved from white marble and arranged as five parallel bridges.',
      },
    },
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
    translations: {
      'en-US': {
        name: 'Five Lantern Lake',
        area: 'Nianhua Bay',
        category: 'Performance',
        description: 'The core lake for Nianhua Bay night tours, hosting the Zen Walk light show and evening visit experience.',
        thumbnail: 'Lake',
        parameters: ['Night water lights', 'Lakeside viewing area', 'High evening visitor flow'],
        culture: 'Light, water, and moving viewing routes create a Zen-inspired night tour atmosphere.',
        highlights: ['Zen Walk light show at 19:00 and 20:00', 'Useful night-tour reminder', 'Good temporary rest stop'],
        openInfo: 'Zen Walk light shows run around 19:00 and 20:00. Arrive at the lakeside early.',
        showTimes: ['19:00 Zen Walk', '20:00 Zen Walk'],
        geofenceIntroText: 'Five Lantern Lake has Zen Walk light shows at 19:00 and 20:00. Arrive at the lakeside early for a better view.',
      },
    },
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
    translations: {
      'en-US': {
        name: 'Xiangfu Temple',
        area: 'Lingshan Scenic Area',
        category: 'Culture & Rest',
        description: 'A traditional temple stop suited for quiet visits, blessings, and historical narration.',
        thumbnail: 'Tem',
        parameters: ['Traditional temple space', 'Blessing visit stop', 'Low-noise area'],
        culture: 'The temple carries Buddhist etiquette, blessing customs, and cultural background.',
        highlights: ['Good for history routes', 'Low-intensity walking', 'Can support vegetarian dining questions'],
        openInfo: 'Open with the scenic area. Please keep quiet during visits.',
        geofenceIntroText: 'Xiangfu Temple is suited for quiet visits and learning about temple etiquette, blessing customs, and Buddhist culture.',
      },
    },
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
    translations: {
      'en-US': {
        name: 'Children Playing with Maitreya',
        area: 'Nianhua Bay',
        category: 'Culture & Rest',
        description: 'A light family stop for interaction, cultural photos, and relaxed visits.',
        thumbnail: 'Kid',
        parameters: ['Family photo spot', 'Interactive sculpture group', 'Low-intensity stay'],
        culture: 'Childlike imagery makes the cultural theme easier for families to approach.',
        highlights: ['Key family route stop', 'Photo stay', 'Pairs well with Brahma Palace performances'],
        openInfo: 'Open all day. Daytime visits are recommended.',
        geofenceIntroText: 'Children Playing with Maitreya is suited for family interaction and photos, making it an easy stop for family visitors.',
      },
    },
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
    translations: {
      'en-US': {
        name: 'Wujinyi Vegetarian Dining',
        area: 'Nianhua Bay',
        category: 'Culture & Rest',
        description: 'A frequent vegetarian dining and rest inquiry point, useful for meal suggestions and recovery breaks.',
        thumbnail: 'Veg',
        parameters: ['Vegetarian dining', 'Rest supply point', 'Before or after night tours'],
        culture: 'Combines vegetarian food culture with a slower touring rhythm, suitable for dining recommendations.',
        highlights: ['Frequent dining questions', 'Rest and supplies', 'Supplementary night-route stop'],
        openInfo: 'Dining hours follow on-site operations.',
      },
    },
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
    translations: {
      'en-US': {
        name: 'History & Culture Route',
        duration: 'About 3 hours',
        summary: 'Highlights the cultural meaning and architectural details of the Grand Buddha, Brahma Palace, Wuming Bridge, and Xiangfu Temple.',
        nodeHighlights: {
          'LS-004': 'Wuming Bridge: five forms of wisdom and white marble bridge details.',
          'LS-001': 'Lingshan Grand Buddha: 101.5m height, 725 tons of copper, and the symbolism of 216 steps.',
          'LS-003': 'Brahma Palace: architectural art and the Auspicious Ode of Lingshan queue reminder.',
          'LS-008': 'Xiangfu Temple: temple etiquette and blessing culture.',
        },
      },
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
    translations: {
      'en-US': {
        name: 'Nature & Scenery Route',
        duration: 'About 2.5 hours',
        summary: 'Keeps narration lighter while focusing on bridges, water views, night touring, and rest rhythm.',
        nodeHighlights: {
          'LS-004': 'Wuming Bridge: crossing symbolizes opening wisdom and works well for photos.',
          'LS-006': 'Nine Dragons Bathing: fountain show and blessing water experience.',
          'NH-005': 'Five Lantern Lake: Zen Walk night light show and lakeside route.',
          'NH-007': 'Wujinyi Vegetarian Dining: vegetarian supplies and night-tour rest.',
        },
      },
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
    translations: {
      'en-US': {
        name: 'Family Route',
        duration: 'About 2 hours',
        summary: 'Prioritizes family-friendly stops such as Nine Dragons Bathing, Children Playing with Maitreya, and Brahma Palace.',
        nodeHighlights: {
          'LS-006': 'Nine Dragons Bathing: clear show times and easy for children to understand.',
          'NH-003': 'Children Playing with Maitreya: interactive photos and a relaxed stop.',
          'LS-003': 'Brahma Palace: indoor visit and interactive performance, useful for rain or heat.',
          'LS-001': 'Lingshan Grand Buddha: short narration around height and step numbers.',
        },
      },
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
    translations: {
      'en-US': {
        title: 'Nine Dragons Bathing Show Reminder',
        message: 'The Nine Dragons Bathing show starts in about 10 minutes. Head over to watch it; blessing water is available after the show.',
      },
    },
  },
  {
    spotId: 'LS-003',
    title: '灵山吉祥颂排队提醒',
    startTime: '14:00',
    advanceMinutes: 30,
    message: '梵宫圣坛需提前排队，建议现在前往。',
    priority: 'medium',
    translations: {
      'en-US': {
        title: 'Auspicious Ode Queue Reminder',
        message: 'The Brahma Palace Sacred Altar requires early queueing. Consider heading there now.',
      },
    },
  },
  {
    spotId: 'NH-005',
    title: '五灯湖夜游提醒',
    startTime: '19:00',
    advanceMinutes: 20,
    message: '五灯湖19:00/20:00有《禅行》灯光秀，湖岸客流较大，建议提前抵达。',
    priority: 'high',
    translations: {
      'en-US': {
        title: 'Five Lantern Lake Night Reminder',
        message: 'Five Lantern Lake has Zen Walk light shows at 19:00 and 20:00. The lakeside gets crowded, so arrive early.',
      },
    },
  },
]

function normalizeLocale(locale: string): LocaleCode {
  return locale === 'en-US' ? 'en-US' : 'zh-CN'
}

function localizeItem<T extends { translations?: Partial<Record<LocaleCode, Partial<T>>> }>(item: T, locale: string): T {
  const translation = item.translations?.[normalizeLocale(locale)]
  return translation ? { ...item, ...translation } : { ...item }
}

export function localizeScenicSpots(locale: string): StructuredScenicSpot[] {
  return SCENIC_SPOTS.map(spot => localizeItem(spot, locale))
}

export function localizeScenicRoutes(locale: string): ScenicRoutePlan[] {
  return SCENIC_ROUTES.map(route => localizeItem(route, locale))
}

export function localizeServiceReminders(locale: string): ServiceReminder[] {
  return SERVICE_REMINDERS.map(reminder => localizeItem(reminder, locale))
}

export function findStructuredSpot(idOrName: string, locale = 'zh-CN') {
  const normalized = idOrName.toLowerCase()
  const spot = SCENIC_SPOTS.find(item => {
    const translatedName = item.translations?.['en-US']?.name?.toLowerCase()
    return item.id.toLowerCase() === normalized
      || item.name.toLowerCase() === normalized
      || translatedName === normalized
  })
  return spot ? localizeItem(spot, locale) : undefined
}

export function buildGuideInsight(text: string, locale = 'zh-CN'): GuideInsight | null {
  const content = text.toLowerCase()
  const hit = SCENIC_SPOTS.find(spot => {
    const translated = spot.translations?.['en-US']
    const terms = [
      spot.id.toLowerCase(),
      spot.name.toLowerCase(),
      translated?.name?.toLowerCase() || '',
      ...spot.parameters.map(item => item.toLowerCase()),
      ...(translated?.parameters || []).map(item => item.toLowerCase()),
    ]
    return terms.some(term => term && content.includes(term))
  })
  if (!hit) return null
  const localized = localizeItem(hit, locale)
  return {
    title: localized.name,
    image: localized.thumbnail,
    tags: [localized.area, localized.category, localized.openInfo].filter(Boolean).slice(0, 3),
    points: [...localized.parameters, ...localized.highlights].slice(0, 5),
  }
}
