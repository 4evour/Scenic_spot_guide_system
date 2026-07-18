const EARTH_RADIUS_M = 6_371_000
const PI = Math.PI
const AXIS = 6378245
const EE = Number('0.00669342162296594323')

export interface GeoCoordinate {
  lat: number
  lng: number
}

export interface EligiblePosition extends GeoCoordinate {
  accuracy: number
}

export interface EligibleSpot extends GeoCoordinate {
  id: string | number
  name: string
  triggerEnabled?: boolean
  triggerRadiusM?: number
}

export function isAccuracyAcceptable(accuracy: number, maxAccuracyM = 10): boolean {
  return Number.isFinite(accuracy) && accuracy >= 0 && accuracy <= maxAccuracyM
}

export function isValidCoordinate(coordinate: GeoCoordinate): boolean {
  return (
    Number.isFinite(coordinate.lat) &&
    Number.isFinite(coordinate.lng) &&
    coordinate.lat >= -90 &&
    coordinate.lat <= 90 &&
    coordinate.lng >= -180 &&
    coordinate.lng <= 180
  )
}

export function haversineDistance(
  lat1: number,
  lng1: number,
  lat2: number,
  lng2: number,
): number {
  const toRad = (degree: number) => (degree * PI) / 180
  const dLat = toRad(lat2 - lat1)
  const dLng = toRad(lng2 - lng1)
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLng / 2) ** 2
  return EARTH_RADIUS_M * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))
}

export function selectClosestEligibleSpot(
  position: EligiblePosition,
  spots: EligibleSpot[],
  options: { maxAccuracyM?: number; defaultRadiusM?: number } = {},
) {
  const { maxAccuracyM = 10, defaultRadiusM = 100 } = options
  if (!isValidCoordinate(position) || !isAccuracyAcceptable(position.accuracy, maxAccuracyM)) {
    return null
  }

  let closest: { spot: EligibleSpot; distanceMeters: number } | null = null
  for (const spot of spots) {
    if (spot.triggerEnabled === false || !isValidCoordinate(spot)) continue
    const distanceMeters = haversineDistance(position.lat, position.lng, spot.lat, spot.lng)
    const radius = spot.triggerRadiusM && spot.triggerRadiusM > 0 ? spot.triggerRadiusM : defaultRadiusM
    if (distanceMeters <= radius && (!closest || distanceMeters < closest.distanceMeters)) {
      closest = { spot, distanceMeters }
    }
  }
  return closest
}

export function wgs84ToGcj02(lng: number, lat: number): GeoCoordinate {
  if (outOfChina(lng, lat)) return { lng, lat }
  let dLat = transformLat(lng - 105, lat - 35)
  let dLng = transformLng(lng - 105, lat - 35)
  const radLat = (lat / 180) * PI
  let magic = Math.sin(radLat)
  magic = 1 - EE * magic * magic
  const sqrtMagic = Math.sqrt(magic)
  dLat = (dLat * 180) / (((AXIS * (1 - EE)) / (magic * sqrtMagic)) * PI)
  dLng = (dLng * 180) / ((AXIS / sqrtMagic) * Math.cos(radLat) * PI)
  return { lng: lng + dLng, lat: lat + dLat }
}

function outOfChina(lng: number, lat: number) {
  return lng < 72.004 || lng > 137.8347 || lat < 0.8293 || lat > 55.8271
}

function transformLat(lng: number, lat: number) {
  let value = -100 + 2 * lng + 3 * lat + 0.2 * lat * lat + 0.1 * lng * lat + 0.2 * Math.sqrt(Math.abs(lng))
  value += ((20 * Math.sin(6 * lng * PI) + 20 * Math.sin(2 * lng * PI)) * 2) / 3
  value += ((20 * Math.sin(lat * PI) + 40 * Math.sin((lat / 3) * PI)) * 2) / 3
  value += ((160 * Math.sin((lat / 12) * PI) + 320 * Math.sin((lat * PI) / 30)) * 2) / 3
  return value
}

function transformLng(lng: number, lat: number) {
  let value = 300 + lng + 2 * lat + 0.1 * lng * lng + 0.1 * lng * lat + 0.1 * Math.sqrt(Math.abs(lng))
  value += ((20 * Math.sin(6 * lng * PI) + 20 * Math.sin(2 * lng * PI)) * 2) / 3
  value += ((20 * Math.sin(lng * PI) + 40 * Math.sin((lng / 3) * PI)) * 2) / 3
  value += ((150 * Math.sin((lng / 12) * PI) + 300 * Math.sin((lng / 30) * PI)) * 2) / 3
  return value
}
