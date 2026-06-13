import { ref, onUnmounted, type Ref } from 'vue';

export interface GeolocationPosition {
  lat: number;
  lng: number;
  accuracy: number; // 精度（米）
  timestamp: number; // 定位时间戳（ms）
}

export interface UseGeolocationOptions {
  enableHighAccuracy?: boolean; // 默认 true
  maximumAge?: number; // 默认 5000（5秒缓存）
  timeout?: number; // 默认 10000（10秒超时）
}

export interface UseGeolocationReturn {
  currentPosition: Ref<GeolocationPosition | null>;
  error: Ref<string | null>;
  isWatching: Ref<boolean>;
  startWatch: () => void;
  stopWatch: () => void;
  permissionGranted: Ref<boolean | null>; // null=未知, true=已授权, false=已拒绝
}

export function useGeolocation(
  options: UseGeolocationOptions = {},
): UseGeolocationReturn {
  const {
    enableHighAccuracy = true,
    maximumAge = 5000,
    timeout = 10000,
  } = options;

  const currentPosition = ref<GeolocationPosition | null>(null);
  const error = ref<string | null>(null);
  const isWatching = ref(false);
  const permissionGranted = ref<boolean | null>(null);

  let watchId: number | null = null;

  function startWatch() {
    if (!navigator.geolocation) {
      error.value = '浏览器不支持地理位置';
      return;
    }

    stopWatch();

    isWatching.value = true;
    error.value = null;

    watchId = navigator.geolocation.watchPosition(
      (pos) => {
        permissionGranted.value = true;
        error.value = null;
        currentPosition.value = {
          lat: pos.coords.latitude,
          lng: pos.coords.longitude,
          accuracy: pos.coords.accuracy,
          timestamp: pos.timestamp,
        };
      },
      (err) => {
        isWatching.value = false;
        if (err.code === err.PERMISSION_DENIED) {
          permissionGranted.value = false;
          error.value = '位置权限被拒绝，请在浏览器设置中允许定位';
        } else if (err.code === err.POSITION_UNAVAILABLE) {
          error.value = '无法获取位置信息，请检查GPS信号';
        } else if (err.code === err.TIMEOUT) {
          error.value = '定位超时';
        } else {
          error.value = `定位失败: ${err.message}`;
        }
      },
      { enableHighAccuracy, maximumAge, timeout },
    );
  }

  function stopWatch() {
    if (watchId !== null) {
      navigator.geolocation.clearWatch(watchId);
      watchId = null;
    }
    isWatching.value = false;
  }

  onUnmounted(() => {
    stopWatch();
  });

  return {
    currentPosition,
    error,
    isWatching,
    startWatch,
    stopWatch,
    permissionGranted,
  };
}
