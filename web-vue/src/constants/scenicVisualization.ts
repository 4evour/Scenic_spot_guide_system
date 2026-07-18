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
  aliases?: string[]
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

export const SCENIC_ROUTE_COORDINATES: Record<string, { id: string; lng: number; lat: number }> = {
  南门: { id: 'LS-R01', lng: 120.102934, lat: 31.420115 },
  灵山大照壁: { id: 'LS-R02', lng: 120.102499, lat: 31.421388 },
  胜境门楼: { id: 'LS-R03', lng: 120.10173, lat: 31.422257 },
  佛足坛: { id: 'LS-R04', lng: 120.101497, lat: 31.422725 },
  五印坛城: { id: 'LS-R05', lng: 120.103054, lat: 31.424676 },
  三圣殿: { id: 'LS-R06', lng: 120.0963, lat: 31.424395 },
  九龙灌浴: { id: 'LS-006', lng: 120.099984, lat: 31.424601 },
  降魔浮雕: { id: 'LS-R07', lng: 120.099569, lat: 31.425559 },
  阿育王柱: { id: 'LS-R08', lng: 120.099261, lat: 31.426188 },
  天下第一掌: { id: 'LS-R09', lng: 120.098366, lat: 31.426957 },
  百子戏弥勒: { id: 'LS-009', lng: 120.098844, lat: 31.42719 },
  灵山蔬食馆: { id: 'LS-R10', lng: 120.100061, lat: 31.426825 },
  祥符禅寺: { id: 'LS-008', lng: 120.098012, lat: 31.427949 },
  杏坛广场: { id: 'LS-R11', lng: 120.097377, lat: 31.428946 },
  灵山大佛: { id: 'LS-001', lng: 120.096477, lat: 31.430194 },
  灵山梵宫: { id: 'LS-003', lng: 120.10242, lat: 31.428218 },
  曼飞龙塔: { id: 'LS-R12', lng: 120.104609, lat: 31.42607 },
  出口: { id: 'LS-R13', lng: 120.105767, lat: 31.421824 },
  文创驿站: { id: 'LS-R14', lng: 120.103651, lat: 31.420196 },
}

const SCENIC_ROUTE_ENGLISH_NAMES: Record<keyof typeof SCENIC_ROUTE_COORDINATES, string> = {
  南门: 'South Gate',
  灵山大照壁: 'Lingshan Screen Wall',
  胜境门楼: 'Scenic Realm Gate Tower',
  佛足坛: 'Buddha Foot Altar',
  五印坛城: 'Five Seal Mandala City',
  三圣殿: 'Three Saints Hall',
  九龙灌浴: 'Nine Dragons Bathing',
  降魔浮雕: 'Demon-Subduing Relief',
  阿育王柱: 'Ashoka Pillar',
  天下第一掌: 'World\'s First Palm',
  百子戏弥勒: 'Children Playing with Maitreya',
  灵山蔬食馆: 'Lingshan Vegetarian Restaurant',
  祥符禅寺: 'Xiangfu Temple',
  杏坛广场: 'Xingtan Square',
  灵山大佛: 'Lingshan Grand Buddha',
  灵山梵宫: 'Brahma Palace',
  曼飞龙塔: 'Manfeilong Pagoda',
  出口: 'Exit',
  文创驿站: 'Cultural Creative Station',
}

function routeOnlySpot(name: keyof typeof SCENIC_ROUTE_COORDINATES, category: string, description: string, thumbnail: string): StructuredScenicSpot {
  const coordinate = SCENIC_ROUTE_COORDINATES[name]
  return {
    id: coordinate.id,
    name,
    area: '灵山胜境',
    category,
    visualType: category.includes('建筑') ? 'landmark' : 'culture',
    description,
    lng: coordinate.lng,
    lat: coordinate.lat,
    rating: 0,
    price: 0,
    imageUrl: '',
    thumbnail,
    parameters: [],
    culture: '',
    highlights: [],
    openInfo: '开放状态以景区现场公告为准。',
    showTimes: [],
    routeTags: [],
    geofenceEnabled: name === '五印坛城',
    geofenceRadiusM: 100,
    geofenceIntroText: '',
    geofenceCooldownMinutes: 1440,
    translations: {
      'en-US': {
        name: SCENIC_ROUTE_ENGLISH_NAMES[name],
        area: 'Lingshan Scenic Area',
        category: 'Route stop',
        description: 'A route stop within Lingshan Scenic Area.',
        openInfo: 'Opening status follows on-site notices.',
      },
    },
  }
}

export const SCENIC_SPOTS: StructuredScenicSpot[] = [
  {
    id: 'LS-001',
    name: '灵山大佛',
    area: '灵山胜境',
    category: '地标建筑',
    visualType: 'landmark',
    description: '灵山胜境核心地标，以大佛主体、登云道和礼佛广场构成强识别游览节点。',
    lng: 120.096477,
    lat: 31.430194,
    rating: 4.9,
    price: 0,
    imageUrl: '',
    thumbnail: '佛',
    parameters: ['高88m', '青铜立佛', '核心礼佛地标'],
    culture: '216级台阶对应108烦恼与108愿望，登临过程强调礼佛与自省。',
    highlights: ['仰观大佛全景', '登云道礼佛动线', '适合数字人讲解高度与台阶寓意'],
    openInfo: '随景区开放，建议上午或傍晚避开强逆光时段。',
    showTimes: [],
    routeTags: ['history', 'family'],
    geofenceEnabled: true,
    geofenceRadiusM: 120,
    geofenceIntroText: '灵山大佛高88米，是灵山胜境的核心礼佛地标。',
    geofenceCooldownMinutes: 180,
    translations: {
      'en-US': {
        name: 'Lingshan Grand Buddha',
        area: 'Lingshan Scenic Area',
        category: 'Landmark',
        description: 'The core landmark of Lingshan, formed by the Buddha statue, Dengyun Path, and worship square.',
        parameters: ['88m tall', 'Bronze standing Buddha', 'Core worship landmark'],
        culture: 'The 216 steps echo 108 worries and 108 wishes, turning the climb into a moment of reflection.',
        highlights: ['Full Buddha panorama', 'Dengyun worship route', 'Height and step symbolism for guided narration'],
        openInfo: 'Open with the scenic area. Visit in the morning or late afternoon to avoid strong backlight.',
        geofenceIntroText: 'The Lingshan Grand Buddha is 88 meters tall and is the core worship landmark of Lingshan Scenic Area.',
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
    lng: 120.099984,
    lat: 31.424601,
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
    name: '灵山梵宫',
    aliases: ['梵宫'],
    area: '灵山胜境',
    category: '地标建筑',
    visualType: 'landmark',
    description: '灵山胜境内建筑规模最大、艺术价值最高的佛教艺术殿堂，兼具文化展陈、佛事活动和圣坛演艺体验。',
    lng: 120.10242,
    lat: 31.428218,
    rating: 4.8,
    price: 0,
    imageUrl: '',
    thumbnail: '宫',
    parameters: ['建筑面积7.2万㎡', '最高处66.5m', '五座莲花圣塔', '圣坛可容纳2000人'],
    culture: '以“莲花藏世界”为设计核心，五座莲花圣塔象征五方五佛，曼陀罗形态圣坛寓意宇宙圆满，是佛教文化与传统艺术融合的代表建筑。',
    highlights: ['星空穹顶与东阳木雕飞天', '琉璃巨制《华藏世界》', '《灵山吉祥颂》沉浸演出', '香水海畔莲花圣塔外观'],
    openInfo: '9:00-17:00开放，冬季闭馆约16:30；《灵山吉祥颂》常见场次为10:35/11:30/14:00/16:00，节假日以现场公告为准。',
    showTimes: ['10:35《灵山吉祥颂》', '11:30《灵山吉祥颂》', '14:00《灵山吉祥颂》', '16:00《灵山吉祥颂》'],
    routeTags: ['history', 'family'],
    geofenceEnabled: true,
    geofenceRadiusM: 110,
    geofenceIntroText: '灵山梵宫建筑面积约7.2万平方米，汇集东阳木雕、琉璃、油画等传统工艺，观看《灵山吉祥颂》建议提前排队。',
    geofenceCooldownMinutes: 180,
    signalBlindSpot: true,
    translations: {
      'en-US': {
        name: 'Brahma Palace',
        area: 'Lingshan Scenic Area',
        category: 'Landmark',
        description: 'An indoor landmark combining architecture, cultural exhibitions, and the Sacred Altar performance.',
        thumbnail: 'Pal',
        parameters: ['72,000 sqm', '66.5m highest point', 'Five lotus towers', 'Sacred Altar for 2,000 viewers'],
        culture: 'Designed around the idea of a lotus-wrapped Buddhist world, its five lotus towers echo the Five Direction Buddhas and its mandala-like altar symbolizes completeness.',
        highlights: ['Starry dome and Dongyang woodcarving apsaras', 'Liuli artwork Huazang World', 'Auspicious Ode of Lingshan performance', 'Lotus towers by Xiangshui Sea'],
        openInfo: 'Usually open 9:00-17:00, around 16:30 in winter. Auspicious Ode is commonly shown at 10:35, 11:30, 14:00, and 16:00; holidays follow on-site notices.',
        showTimes: ['10:35 Auspicious Ode', '11:30 Auspicious Ode', '14:00 Auspicious Ode', '16:00 Auspicious Ode'],
        geofenceIntroText: 'Brahma Palace covers about 72,000 sqm and features Dongyang woodcarving, liuli, paintings, and other traditional crafts. Queue early for Auspicious Ode of Lingshan.',
      },
    },
  },
  {
    id: 'LS-R14',
    name: '文创驿站',
    area: '灵山胜境',
    category: '文化休憩',
    visualType: 'culture',
    description: '提供文创商品、饮品和游客休憩服务。',
    lng: 120.103651,
    lat: 31.420196,
    rating: 4.5,
    price: 0,
    imageUrl: '',
    thumbnail: '创',
    parameters: ['文创商品', '饮品补给', '游客休憩'],
    culture: '为游览过程提供文创与休憩服务。',
    highlights: ['适合行程末段补给', '文创纪念品咨询', '开放状态以现场为准'],
    openInfo: '营业时间以现场公告为准。',
    showTimes: [],
    routeTags: ['family'],
    geofenceEnabled: false,
    geofenceRadiusM: 80,
    geofenceIntroText: '',
    geofenceCooldownMinutes: 1440,
    translations: {
      'en-US': {
        name: 'Cultural Creative Station',
        area: 'Lingshan Scenic Area',
        category: 'Culture & Rest',
        description: 'A rest stop for cultural creative products and drinks.',
        thumbnail: 'CC',
        parameters: ['Cultural creative products', 'Drinks', 'Visitor rest'],
        culture: 'Provides cultural creative and rest services during a visit.',
        highlights: ['Late-route supplies', 'Souvenir consultation', 'On-site hours apply'],
        openInfo: 'Hours follow on-site notices.',
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
    lng: 120.098012,
    lat: 31.427949,
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
    geofenceEnabled: false,
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
    id: 'LS-009',
    name: '百子戏弥勒',
    area: '灵山胜境',
    category: '文化休憩',
    visualType: 'culture',
    description: '适合亲子互动和文化拍照的轻量游览点。',
    lng: 120.098844,
    lat: 31.42719,
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
    geofenceEnabled: false,
    geofenceRadiusM: 90,
    geofenceIntroText: '百子戏弥勒适合亲子互动和拍照，可作为家庭游客的轻松停留点。',
    geofenceCooldownMinutes: 180,
    translations: {
      'en-US': {
        name: 'Children Playing with Maitreya',
        area: 'Lingshan Scenic Area',
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
  routeOnlySpot('南门', '出入口', '景区步行路线的主要入口。', '门'),
  routeOnlySpot('灵山大照壁', '文化景观', '位于景区入口区域的标志性照壁。', '壁'),
  routeOnlySpot('胜境门楼', '文化景观', '连接入口与景区主游线的门楼节点。', '楼'),
  routeOnlySpot('佛足坛', '文化景观', '以佛足印文化为主题的游览节点。', '足'),
  routeOnlySpot('五印坛城', '文化建筑', '展示藏传佛教文化的建筑景观。', '坛'),
  routeOnlySpot('三圣殿', '文化建筑', '景区佛教文化建筑节点。', '殿'),
  routeOnlySpot('降魔浮雕', '文化景观', '讲述释迦牟尼降伏心魔、彻悟成佛故事的浮雕景观。', '雕'),
  routeOnlySpot('阿育王柱', '文化景观', '以四方狮子等佛教文化意象构成的石柱景观。', '柱'),
  routeOnlySpot('天下第一掌', '文化景观', '按灵山大佛右手复制的佛手文化景观。', '掌'),
  routeOnlySpot('灵山蔬食馆', '餐饮服务', '景区内提供蔬食餐饮服务的设施。', '食'),
  routeOnlySpot('杏坛广场', '游览节点', '连接祥符禅寺与灵山大佛游线的广场节点。', '场'),
  routeOnlySpot('曼飞龙塔', '文化建筑', '展示南传佛教建筑风格的塔群景观。', '塔'),
  routeOnlySpot('出口', '出入口', '景区路线结束和离园节点。', '出'),
]

export const SCENIC_ROUTES: ScenicRoutePlan[] = [
  {
    id: 'history',
    name: '历史文化路线',
    duration: '约3小时',
    summary: '突出佛足坛、九龙灌浴、梵宫、大佛和祥符禅寺的文化内涵。',
    spotIds: ['LS-R04', 'LS-006', 'LS-003', 'LS-001', 'LS-008'],
    nodeHighlights: {
      'LS-R04': '佛足坛：从佛足印切入佛教文化讲解。',
      'LS-006': '九龙灌浴：动态音乐群雕与佛诞故事。',
      'LS-003': '梵宫：建筑艺术与《灵山吉祥颂》观演提醒。',
      'LS-001': '灵山大佛：高88米的青铜立佛。',
      'LS-008': '祥符禅寺：寺院礼仪与祈福文化。',
    },
    translations: {
      'en-US': {
        name: 'History & Culture Route',
        duration: 'About 3 hours',
        summary: 'Highlights Buddhist culture at the Buddha Foot Altar, Nine Dragons Bathing, Brahma Palace, Grand Buddha, and Xiangfu Temple.',
        nodeHighlights: {
          'LS-R04': 'Buddha Foot Altar: an entry point for Buddhist cultural narration.',
          'LS-006': 'Nine Dragons Bathing: a dynamic musical sculpture presenting the Buddhist birth story.',
          'LS-003': 'Brahma Palace: architectural art and the Auspicious Ode of Lingshan queue reminder.',
          'LS-001': 'Lingshan Grand Buddha: an 88-meter bronze standing Buddha.',
          'LS-008': 'Xiangfu Temple: temple etiquette and blessing culture.',
        },
      },
    },
  },
  {
    id: 'nature',
    name: '自然风光路线',
    duration: '约2.5小时',
    summary: '降低高密度讲解，突出入口景观、喷泉体验和蔬食休憩节奏。',
    spotIds: ['LS-R02', 'LS-006', 'LS-R08', 'LS-R10'],
    nodeHighlights: {
      'LS-R02': '灵山大照壁：入口形象和主游线起点。',
      'LS-006': '九龙灌浴：喷泉表演与祈福圣水体验。',
      'LS-R08': '阿育王柱：开阔广场上的文化景观。',
      'LS-R10': '灵山蔬食馆：蔬食补给和行程休憩。',
    },
    translations: {
      'en-US': {
        name: 'Nature & Scenery Route',
        duration: 'About 2.5 hours',
        summary: 'Keeps narration lighter while focusing on entrance scenery, the fountain experience, and a vegetarian rest stop.',
        nodeHighlights: {
          'LS-R02': 'Lingshan Screen Wall: the entrance landmark and start of the main route.',
          'LS-006': 'Nine Dragons Bathing: fountain show and blessing water experience.',
          'LS-R08': 'Ashoka Pillar: a cultural landmark in an open square.',
          'LS-R10': 'Lingshan Vegetarian Restaurant: food supplies and a rest stop.',
        },
      },
    },
  },
  {
    id: 'family',
    name: '亲子路线',
    duration: '约2小时',
    summary: '优先体验九龙灌浴、百子戏弥勒、梵宫等亲子友好点位。',
    spotIds: ['LS-006', 'LS-009', 'LS-003', 'LS-001'],
    nodeHighlights: {
      'LS-006': '九龙灌浴：固定演出时段清晰，孩子容易理解。',
      'LS-009': '百子戏弥勒：互动拍照和轻松停留。',
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
          'LS-009': 'Children Playing with Maitreya: interactive photos and a relaxed stop.',
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
    const aliases = item.aliases?.map(alias => alias.toLowerCase()) || []
    return item.id.toLowerCase() === normalized
      || item.name.toLowerCase() === normalized
      || aliases.includes(normalized)
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
