import { ref, watch, type Ref } from 'vue';
import type { GeolocationPosition } from './useGeolocation';
import {
  isAccuracyAcceptable,
  isValidCoordinate,
  selectClosestEligibleSpot,
} from '../utils/geolocation.ts';

const REQUIRED_STABLE_SAMPLES = 3;
const MAX_SAMPLE_WINDOW = 5;

export interface SpotWithCoords {
  id: string | number;
  name: string;
  lat: number;
  lng: number;
  triggerEnabled?: boolean;
  triggerRadiusM?: number;
  introText?: string;
  cooldownMinutes?: number;
}

export interface UseProximityGuideOptions {
  /** 触发半径（米），默认 100m */
  triggerRadiusM?: number;
  /** 允许自动触发的设备定位误差上限（米），默认 10m */
  maxAccuracyM?: number;
  /** 是否允许本次位置消耗景点触发机会 */
  canTrigger?: Ref<boolean>;
  /** 跨页面本地冷却记录 key */
  storageKey?: string;
}

export interface UseProximityGuideReturn {
  /** 当前进入触发半径的景点（仅当位置变化且进入新景点范围时更新） */
  nearbySpot: Ref<SpotWithCoords | null>;
  /** 已触发过的景点 ID 集合 */
  triggeredSpots: Ref<Set<string | number>>;
  /** 重置已触发记录（关闭自动导览时调用） */
  resetTriggered: () => void;
  /** 注入景点坐标列表 */
  setSpots: (spots: SpotWithCoords[]) => void;
}

export function useProximityGuide(
  currentPosition: Ref<GeolocationPosition | null>,
  options: UseProximityGuideOptions = {},
): UseProximityGuideReturn {
  const {
    triggerRadiusM = 100,
    maxAccuracyM = 10,
    canTrigger,
    storageKey = 'sg_geofence_triggered_at',
  } = options;

  const spots = ref<SpotWithCoords[]>([]);
  const nearbySpot = ref<SpotWithCoords | null>(null);
  const triggeredSpots = ref<Set<string | number>>(new Set());
  const positionSamples: GeolocationPosition[] = [];
  let lastSampleTimestamp: number | null = null;

  function setSpots(newSpots: SpotWithCoords[]) {
    spots.value = newSpots;
    evaluatePosition(currentPosition.value);
  }

  function resetTriggered() {
    triggeredSpots.value = new Set();
    nearbySpot.value = null;
    positionSamples.length = 0;
    lastSampleTimestamp = null;
  }

  function evaluatePosition(pos: GeolocationPosition | null) {
    if (!pos) {
      nearbySpot.value = null;
      return;
    }

    const stablePosition = updateStablePosition(pos);
    if (!stablePosition || spots.value.length === 0 || (canTrigger && !canTrigger.value)) {
      nearbySpot.value = null;
      return;
    }

    const eligibleSpots = spots.value.filter(
      (spot) => !triggeredSpots.value.has(spot.id) && !isCoolingDown(spot, storageKey),
    );
    const closest = selectClosestEligibleSpot(stablePosition, eligibleSpots, {
      maxAccuracyM,
      defaultRadiusM: triggerRadiusM,
    });

    if (closest) {
      // 标记为已触发
      triggeredSpots.value.add(closest.spot.id);
      markTriggered(closest.spot, storageKey);
      nearbySpot.value = closest.spot;
    } else {
      nearbySpot.value = null;
    }
  }

  function updateStablePosition(pos: GeolocationPosition) {
    if (!isValidCoordinate(pos) || !isAccuracyAcceptable(pos.accuracy, maxAccuracyM)) {
      return null;
    }
    if (pos.timestamp !== lastSampleTimestamp) {
      positionSamples.push(pos);
      if (positionSamples.length > MAX_SAMPLE_WINDOW) positionSamples.shift();
      lastSampleTimestamp = pos.timestamp;
    }
    if (positionSamples.length < REQUIRED_STABLE_SAMPLES) return null;

    const recent = positionSamples.slice(-REQUIRED_STABLE_SAMPLES);
    return {
      lat: median(recent.map((sample) => sample.lat)),
      lng: median(recent.map((sample) => sample.lng)),
      accuracy: median(recent.map((sample) => sample.accuracy)),
      timestamp: recent[recent.length - 1].timestamp,
    };
  }

  watch(currentPosition, evaluatePosition);
  if (canTrigger) watch(canTrigger, () => evaluatePosition(currentPosition.value));

  return {
    nearbySpot,
    triggeredSpots,
    resetTriggered,
    setSpots,
  };
}

function median(values: number[]) {
  const [first, second, third] = values;
  if ((first <= second && second <= third) || (third <= second && second <= first)) return second;
  if ((second <= first && first <= third) || (third <= first && first <= second)) return first;
  return third;
}

function readTriggered(storageKey: string): Record<string, number> {
  try {
    return JSON.parse(localStorage.getItem(storageKey) || '{}') as Record<string, number>;
  } catch {
    return {};
  }
}

function markTriggered(spot: SpotWithCoords, storageKey: string) {
  try {
    const data = readTriggered(storageKey);
    data[String(spot.id)] = Date.now();
    localStorage.setItem(storageKey, JSON.stringify(data));
  } catch {
    // Keep in-memory triggering usable when browser storage is unavailable.
  }
}

function isCoolingDown(spot: SpotWithCoords, storageKey: string) {
  const cooldownMinutes = spot.cooldownMinutes && spot.cooldownMinutes > 0 ? spot.cooldownMinutes : 1440;
  const last = readTriggered(storageKey)[String(spot.id)];
  return Boolean(last && Date.now() - last < cooldownMinutes * 60 * 1000);
}
