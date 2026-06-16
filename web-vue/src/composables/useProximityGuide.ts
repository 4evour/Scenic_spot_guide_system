import { ref, watch, type Ref } from 'vue';
import type { GeolocationPosition } from './useGeolocation';

const EARTH_RADIUS_M = 6_371_000;

/**
 * 使用 Haversine 公式计算两个经纬度坐标之间的球面距离（单位：米）。
 */
export function haversineDistance(
  lat1: number,
  lng1: number,
  lat2: number,
  lng2: number,
): number {
  const toRad = (deg: number) => (deg * Math.PI) / 180;
  const dLat = toRad(lat2 - lat1);
  const dLng = toRad(lng2 - lng1);
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(toRad(lat1)) *
      Math.cos(toRad(lat2)) *
      Math.sin(dLng / 2) ** 2;
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
  return EARTH_RADIUS_M * c;
}

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
  const { triggerRadiusM = 100, storageKey = 'sg_geofence_triggered_at' } = options;

  const spots = ref<SpotWithCoords[]>([]);
  const nearbySpot = ref<SpotWithCoords | null>(null);
  const triggeredSpots = ref<Set<string | number>>(new Set());

  function setSpots(newSpots: SpotWithCoords[]) {
    spots.value = newSpots;
  }

  function resetTriggered() {
    triggeredSpots.value = new Set();
    nearbySpot.value = null;
  }

  watch(currentPosition, (pos) => {
    if (!pos || spots.value.length === 0) {
      nearbySpot.value = null;
      return;
    }

    let closest: { spot: SpotWithCoords; dist: number } | null = null;

    for (const spot of spots.value) {
      if (spot.triggerEnabled === false) continue;
      // 跳过已触发的景点（每页生命周期内只触发一次）
      if (triggeredSpots.value.has(spot.id)) continue;
      if (isCoolingDown(spot, storageKey)) continue;

      const dist = haversineDistance(pos.lat, pos.lng, spot.lat, spot.lng);
      const radius = spot.triggerRadiusM && spot.triggerRadiusM > 0 ? spot.triggerRadiusM : triggerRadiusM;
      if (dist <= radius) {
        if (!closest || dist < closest.dist) {
          closest = { spot, dist };
        }
      }
    }

    if (closest) {
      // 标记为已触发
      triggeredSpots.value.add(closest.spot.id);
      markTriggered(closest.spot, storageKey);
      nearbySpot.value = closest.spot;
    } else {
      nearbySpot.value = null;
    }
  });

  return {
    nearbySpot,
    triggeredSpots,
    resetTriggered,
    setSpots,
  };
}

function readTriggered(storageKey: string): Record<string, number> {
  try {
    return JSON.parse(localStorage.getItem(storageKey) || '{}') as Record<string, number>;
  } catch {
    return {};
  }
}

function markTriggered(spot: SpotWithCoords, storageKey: string) {
  const data = readTriggered(storageKey);
  data[String(spot.id)] = Date.now();
  localStorage.setItem(storageKey, JSON.stringify(data));
}

function isCoolingDown(spot: SpotWithCoords, storageKey: string) {
  const cooldownMinutes = spot.cooldownMinutes && spot.cooldownMinutes > 0 ? spot.cooldownMinutes : 1440;
  const last = readTriggered(storageKey)[String(spot.id)];
  return Boolean(last && Date.now() - last < cooldownMinutes * 60 * 1000);
}
