import { ref, onUnmounted, type Ref } from 'vue';
import { wgs84ToGcj02 } from '../utils/geolocation.ts';

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
  messages?: Partial<GeolocationMessages>;
}

export interface GeolocationMessages {
  notSupported: () => string;
  denied: () => string;
  unavailable: () => string;
  timeout: () => string;
  failed: (message: string) => string;
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
    messages = {},
  } = options;

  const defaultMessages: GeolocationMessages = {
    notSupported: () => 'Geolocation is not supported in this browser',
    denied: () => 'Location permission denied',
    unavailable: () => 'Location is unavailable',
    timeout: () => 'Location request timed out',
    failed: (message) => `Location failed: ${message}`,
  };
  const geolocationMessages = { ...defaultMessages, ...messages };

  const currentPosition = ref<GeolocationPosition | null>(null);
  const error = ref<string | null>(null);
  const isWatching = ref(false);
  const permissionGranted = ref<boolean | null>(null);

  let watchId: number | null = null;

  function startWatch() {
    if (!navigator.geolocation) {
      error.value = geolocationMessages.notSupported();
      return;
    }

    stopWatch();

    isWatching.value = true;
    error.value = null;

    watchId = navigator.geolocation.watchPosition(
      (pos) => {
        permissionGranted.value = true;
        error.value = null;
        const converted = wgs84ToGcj02(pos.coords.longitude, pos.coords.latitude);
        currentPosition.value = {
          lat: converted.lat,
          lng: converted.lng,
          accuracy: pos.coords.accuracy,
          timestamp: pos.timestamp,
        };
      },
      (err) => {
        isWatching.value = false;
        if (err.code === err.PERMISSION_DENIED) {
          permissionGranted.value = false;
          error.value = geolocationMessages.denied();
        } else if (err.code === err.POSITION_UNAVAILABLE) {
          error.value = geolocationMessages.unavailable();
        } else if (err.code === err.TIMEOUT) {
          error.value = geolocationMessages.timeout();
        } else {
          error.value = geolocationMessages.failed(err.message);
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
