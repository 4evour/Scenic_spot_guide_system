import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { nextTick, ref } from 'vue'
import { SCENIC_ROUTE_COORDINATES, SCENIC_SPOTS } from '../src/constants/scenicVisualization.ts'
import {
  haversineDistance,
  isAccuracyAcceptable,
  selectClosestEligibleSpot,
  wgs84ToGcj02,
} from '../src/utils/geolocation.ts'
import { useProximityGuide } from '../src/composables/useProximityGuide.ts'

const calibration = JSON.parse(
  readFileSync(new URL('../../configs/scenic_spot_coordinates.json', import.meta.url), 'utf8'),
)
assert.equal(calibration.spots.length, 20)
assert.equal(Object.keys(SCENIC_ROUTE_COORDINATES).length, 20)
for (const expected of calibration.spots) {
  const coordinate = SCENIC_ROUTE_COORDINATES[expected.name]
  assert.ok(coordinate, `missing route coordinate for ${expected.name}`)
  assert.equal(coordinate.lng, expected.longitude)
  assert.equal(coordinate.lat, expected.latitude)
  const scenicSpot = SCENIC_SPOTS.find((item) => item.name === expected.name)
  assert.ok(scenicSpot, `missing fallback scenic spot for ${expected.name}`)
  assert.equal(scenicSpot.lng, expected.longitude)
  assert.equal(scenicSpot.lat, expected.latitude)
  assert.equal(scenicSpot.geofenceEnabled, expected.verified && expected.geofence_enabled)
}

const beijing = wgs84ToGcj02(116.404, 39.915)
assert.ok(Math.abs(beijing.lng - 116.410244) < 0.00001)
assert.ok(Math.abs(beijing.lat - 39.916404) < 0.00001)

assert.equal(isAccuracyAcceptable(10), true)
assert.equal(isAccuracyAcceptable(10.01), false)
assert.equal(isAccuracyAcceptable(Number.NaN), false)

const spot = {
  id: 'LS-001',
  name: '灵山大佛',
  lat: SCENIC_ROUTE_COORDINATES['灵山大佛'].lat,
  lng: SCENIC_ROUTE_COORDINATES['灵山大佛'].lng,
  triggerRadiusM: 100,
  triggerEnabled: true,
}

assert.equal(
  selectClosestEligibleSpot(
    { lat: spot.lat, lng: spot.lng, accuracy: 9 },
    [spot],
    { maxAccuracyM: 10 },
  )?.spot.id,
  'LS-001',
)
assert.equal(
  selectClosestEligibleSpot(
    { lat: spot.lat, lng: spot.lng, accuracy: 11 },
    [spot],
    { maxAccuracyM: 10 },
  ),
  null,
)

const storage = new Map()
globalThis.localStorage = {
  getItem: (key) => storage.get(key) ?? null,
  setItem: (key, value) => storage.set(key, value),
  removeItem: (key) => storage.delete(key),
}

const boundaryPosition = { lat: spot.lat, lng: spot.lng + 0.001, accuracy: 9 }
const boundaryDistance = haversineDistance(
  boundaryPosition.lat,
  boundaryPosition.lng,
  spot.lat,
  spot.lng,
)
assert.equal(
  selectClosestEligibleSpot(boundaryPosition, [{ ...spot, triggerRadiusM: boundaryDistance }])?.spot.id,
  spot.id,
)
assert.equal(
  selectClosestEligibleSpot(boundaryPosition, [{ ...spot, triggerRadiusM: boundaryDistance - 0.001 }]),
  null,
)

async function submitSamples(positionRef, samples) {
  for (const sample of samples) {
    positionRef.value = sample
    await nextTick()
  }
}

function samplesAt(target, startTimestamp = Date.now()) {
  return [0, 1, 2].map((index) => ({
    lat: target.lat,
    lng: target.lng,
    accuracy: target.accuracy ?? 9,
    timestamp: startTimestamp + index,
  }))
}

const currentPosition = ref(null)
const proximity = useProximityGuide(currentPosition, {
  storageKey: `test-geofence-${Date.now()}`,
  maxAccuracyM: 10,
})
proximity.setSpots([spot])
await submitSamples(currentPosition, [
  { lat: spot.lat, lng: spot.lng, accuracy: 11, timestamp: Date.now() },
])
await nextTick()
assert.equal(proximity.nearbySpot.value, null)

const stablePosition = ref(null)
const stableProximity = useProximityGuide(stablePosition, {
  storageKey: `test-geofence-stable-${Date.now()}`,
  maxAccuracyM: 10,
})
stableProximity.setSpots([spot])
const stableSamples = samplesAt({ lat: spot.lat, lng: spot.lng, accuracy: 9 })
await submitSamples(stablePosition, stableSamples.slice(0, 1))
assert.equal(stableProximity.nearbySpot.value, null)
await submitSamples(stablePosition, stableSamples.slice(1, 2))
assert.equal(stableProximity.nearbySpot.value, null)
await submitSamples(stablePosition, stableSamples.slice(2))
assert.equal(stableProximity.nearbySpot.value?.id, 'LS-001')

const positionReadyBeforeSpots = ref(null)
const lateSpotsProximity = useProximityGuide(positionReadyBeforeSpots, {
  storageKey: `test-geofence-late-spots-${Date.now()}`,
  maxAccuracyM: 10,
})
await submitSamples(positionReadyBeforeSpots, samplesAt({ lat: spot.lat, lng: spot.lng, accuracy: 9 }))
lateSpotsProximity.setSpots([spot])
await nextTick()
assert.equal(lateSpotsProximity.nearbySpot.value?.id, 'LS-001')

const cooldownStorageKey = `test-geofence-cooldown-${Date.now()}`
const firstVisitPosition = ref(null)
const firstVisit = useProximityGuide(firstVisitPosition, { storageKey: cooldownStorageKey })
firstVisit.setSpots([spot])
await submitSamples(firstVisitPosition, samplesAt({ lat: spot.lat, lng: spot.lng, accuracy: 9 }))
assert.equal(firstVisit.nearbySpot.value?.id, 'LS-001')
firstVisit.resetTriggered()

const returnVisitPosition = ref(null)
const returnVisit = useProximityGuide(returnVisitPosition, { storageKey: cooldownStorageKey })
returnVisit.setSpots([spot])
await submitSamples(returnVisitPosition, samplesAt({ lat: spot.lat, lng: spot.lng, accuracy: 9 }))
assert.equal(returnVisit.nearbySpot.value, null)

const canTrigger = ref(false)
const lockedPosition = ref(null)
const lockedGuide = useProximityGuide(lockedPosition, {
  storageKey: `test-geofence-audio-${Date.now()}`,
  canTrigger,
})
lockedGuide.setSpots([spot])
await submitSamples(lockedPosition, samplesAt({ lat: spot.lat, lng: spot.lng, accuracy: 9 }))
assert.equal(lockedGuide.nearbySpot.value, null)
assert.equal(lockedGuide.triggeredSpots.value.size, 0)
canTrigger.value = true
await nextTick()
assert.equal(lockedGuide.nearbySpot.value?.id, 'LS-001')

const demoPosition = ref(null)
const demoGuide = useProximityGuide(demoPosition, {
  storageKey: `test-geofence-demo-${Date.now()}`,
})
demoGuide.setSpots([spot])
await submitSamples(demoPosition, samplesAt({ lat: spot.lat, lng: spot.lng, accuracy: 0 }))
assert.equal(demoGuide.nearbySpot.value?.id, 'LS-001')

console.log('geolocation tests passed')
