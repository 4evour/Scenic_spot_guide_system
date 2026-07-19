# Scenic Guide API

Base URL: `/api/v1`

All responses from project handlers use:

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

Protected endpoints require an authenticated session. Browser clients should use
the HttpOnly `auth_token` cookie set by `/login`; non-browser clients may still
send `Authorization: Bearer <token>` for API compatibility.

```http
Authorization: Bearer <token>
```

Public JSON fields use `snake_case`. For example, guide content exposes
`content_type` and `audio_url`; scenic spots expose `image_url`, `sort_order`,
`created_at`, and `updated_at`.

## Auth

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/register` | Public, rate limited | Register a visitor account. Client supplied roles are ignored. |
| POST | `/login` | Public | Login and set the `auth_token` HttpOnly Cookie. The response body returns user profile data, not a token. |
| POST | `/logout` | User | Clear the `auth_token` Cookie. |
| GET | `/user/me` | User | Read the current Cookie-backed session. |
| PUT | `/user/password` | User | Change the current registered user's password. Guests must upgrade before changing password. |
| GET | `/users/:id` | User | Read an owned user profile; admins can read any profile. |
| PUT | `/users/:id` | User | Update an owned user profile; admins can update any profile. |
| DELETE | `/users/:id` | User | Delete an owned user profile; admins can delete any profile. |
| POST | `/auth/guest-login` | Public, rate limited | Create or restore a guest session from a device fingerprint and set cookies. |
| POST | `/auth/upgrade-guest` | User | Upgrade a guest account to a registered visitor account. |
| GET | `/demo-info` | Public, local demo only | Return reviewer accounts only when local demo mode is explicitly enabled and the direct client is a loopback address. Otherwise returns `enabled: false`. Responses are `no-store`. |
| GET | `/user/avatar-preference` | User | Read the current user's preferred Live2D avatar ID. |
| PUT | `/user/avatar-preference` | User | Update the preferred avatar ID. Supported values are validated server-side. |

## Scenic Spots

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/spots` | Public | List scenic spots. |
| GET | `/spots/:id` | Public | Get one scenic spot. |
| GET | `/spots/category` | Public | List spots by category. |
| GET | `/spots/nearby?lat=...&lng=...&radius=500` | Public | List nearby spots using configured coordinates and geofence radius. |
| POST | `/spots` | Admin | Create a scenic spot. |
| PUT | `/spots/:id` | Admin | Update a scenic spot. |
| DELETE | `/spots/:id` | Admin | Delete a scenic spot. |

## Guide Content

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/contents?page=1&page_size=20` | Admin | Paginated admin list for guide content management. |
| GET | `/contents/:id` | Public | Get one guide content item. |
| GET | `/contents/spot/:spot_id` | Public | List guide content by scenic spot. |
| GET | `/contents/spot/:spot_id/type?type=...` | Public | List guide content by scenic spot and content type. |
| POST | `/contents` | Admin | Create guide content. |
| PUT | `/contents/:id` | Admin | Update guide content. |
| DELETE | `/contents/:id` | Admin | Delete guide content. |

## Tour Routes

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/routes` | Public | List routes. |
| GET | `/routes/:id` | Public | Get one route. |
| GET | `/routes/difficulty?difficulty=...` | Public | List routes by difficulty. |
| POST | `/routes` | Admin | Create a route. |
| PUT | `/routes/:id` | Admin | Update a route. |
| DELETE | `/routes/:id` | Admin | Delete a route. |

## Scenic Profile

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/scenic/profile` | Public | Read the active scenic profile, including digital human settings, quick asks, routes, and topic entities. |
| GET | `/scenic/quick-asks` | Public | List generated quick questions for the active scenic profile. |
| GET | `/scenic/persona` | Public | Read the digital human persona prompt and voice settings generated from the active scenic profile. |

## Visitor Queries

Vue 管理端的“游客问题”页面已接入以下管理员接口，支持全部/未回答切换、回复编辑、处理状态更新和删除。请求/响应字段使用 `query`、`response`、`spot_id`、`is_answered`、`created_at`。

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/queries` | User | Create a visitor query. |
| GET | `/queries/:id` | User | Get one query. |
| GET | `/queries` | Admin | List queries. |
| GET | `/queries/unanswered` | Admin | List unanswered queries. |
| PUT | `/queries/:id` | Admin | Update a query. |
| DELETE | `/queries/:id` | Admin | Delete a query. |

## AI And Knowledge

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/ai/chat` | Public, optional user | Ask the RAG chat service. Browser requests may be auto-bound to a guest session and can pass `session_id` for short-term context. |
| POST | `/ai/feedback` | Public, optional user | Submit feedback for a chat answer. Browser requests may be auto-bound to a guest session. |
| POST | `/ai/tts` | Public | Return MP3 audio synthesized by the configured TTS provider. |
| POST | `/ai/tts/stream` | Public | Stream synthesized MP3 chunks as they arrive. |
| GET | `/knowledge/list` | Admin | List knowledge chunks with database pagination and filters. |
| GET | `/knowledge/:id` | Admin | Get one knowledge chunk. |
| POST | `/knowledge` | Admin | Create a knowledge chunk. |
| POST | `/knowledge/upload` | Admin | Upload JSONL, JSON, Markdown, or TXT, max 10 MB. |
| PUT | `/knowledge/:id` | Admin | Update a knowledge chunk. |
| DELETE | `/knowledge/:id` | Admin | Delete a knowledge chunk. |
| DELETE | `/knowledge/all` | Admin | Delete all knowledge chunks. Body must include `{"confirm":"DELETE_ALL_KNOWLEDGE"}`. |

## Sessions

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/sessions?page=1&page_size=20` | User | List the current user's persisted chat sessions. |
| GET | `/sessions/:session_id/messages?limit=50&before_id=0` | User | List messages in an owned session. Unknown local sessions return an empty list. |
| POST | `/sessions/:session_id/messages` | User | Persist one message. Body: `role`, `content`, optional `emotion`, `response_time_ms`. Creates a session for the current user if needed; rejects writes to another user's session. |
| DELETE | `/sessions/:session_id` | User | Delete an owned session and its messages. |
| GET | `/sessions/search?keyword=...&page=1&page_size=20` | User | Search the current user's historical messages. |

## QR Guide

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/qr/:code` | Public | Resolve an enabled scenic spot QR code to spot metadata. Disabled or unknown codes return 404. |
| POST | `/qr/:code/intro` | Public | Resolve an enabled QR code and return cached or generated digital-human intro text plus follow-up questions. |
| GET | `/admin/qr/spots` | Admin | List scenic spots with QR code, intro text, and enabled status. |
| PUT | `/admin/qr/spots/:id` | Admin | Update one spot's `qr_code`, `qr_intro_text`, and `qr_enabled`. Old and new intro caches are invalidated. |
| GET | `/admin/qr/spots/:id/image?format=png\|svg` | Admin | Download a PNG or SVG QR image that points to `/scan?id=<qr_code>`. |
| POST | `/admin/qr/batch-generate` | Admin | Generate missing QR IDs for spots that do not have one. |
| GET | `/admin/qr/stats` | Admin | Read QR counts and intro cache status. |

## Digital Human

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/digital-human/avatar-options` | Public | List supported Live2D avatars. Admin configuration may restrict the list to the default avatar. |
| POST | `/dh/session/create` | User | Create a short-lived digital human session ID. |
| POST | `/dh/chat/text` | User | Text chat with emotion-tagged answer. |
| POST | `/dh/chat/voice-transcript` | User | Chat using a speech transcript. |
| POST | `/dh/feedback` | User | Submit feedback. |
| GET | `/dh/health` | Public | Digital human API health. |

## Admin

All `/admin/*` endpoints require admin authentication.

| Method | Path | Description |
| --- | --- | --- |
| GET | `/admin/users?page=1&page_size=20` | Paginated user list. |
| GET | `/admin/users/role?role=...` | List users by role. |
| POST | `/admin/users` | Create a user. Password policy and bcrypt hashing are enforced server-side. |
| PUT | `/admin/users/:id` | Update username, email, role, and optionally password. Empty password keeps the existing hash. |
| DELETE | `/admin/users/:id` | Delete a user. Missing users return 404. |
| GET | `/admin/dashboard/overview` | Dashboard overview. |
| GET | `/admin/dashboard/hourly-trend` | 24-hour trend. |
| GET | `/admin/dashboard/top-questions` | Top questions. |
| GET | `/admin/dashboard/category-distribution` | Category distribution. |
| GET | `/admin/dashboard/response-time-distribution` | Response time distribution. |
| GET | `/admin/dashboard/satisfaction-trend` | Satisfaction trend. |
| GET | `/admin/dashboard/recent-conversations` | Recent conversations. |
| GET | `/admin/reports/visitor?period=7d\|30d` | Visitor report for the selected reporting window. Empty periods return empty chart data plus no-data suggestions instead of fabricated demo metrics. |
| GET | `/admin/digital-human/config` | Digital human configuration. |
| PUT | `/admin/digital-human/config` | Update digital human configuration. |
| GET | `/admin/settings` | System settings. |
| PUT | `/admin/settings` | Update system settings. |
| GET | `/admin/knowledge/stats` | Knowledge base statistics. |
| GET | `/admin/knowledge/eval-stats` | RAG evaluation report data when available. |
| GET | `/admin/qr/spots` | List scenic spots with QR configuration. |
| PUT | `/admin/qr/spots/:id` | Update a scenic spot's QR configuration. |
| GET | `/admin/qr/spots/:id/image?format=png\|svg` | Download a generated QR image. |
| POST | `/admin/qr/batch-generate` | Generate missing QR IDs in bulk. |
| GET | `/admin/qr/stats` | QR management statistics. |
| POST | `/admin/insights/sessions/:session_id/analyze` | Analyze a persisted visitor session and generate satisfaction insights. |
| GET | `/admin/insights/analyses` | List AI-generated visitor insight analyses. |
| GET | `/admin/knowledge/candidates?status=...` | List knowledge candidates generated from visitor insights. |
| POST | `/admin/knowledge/candidates/:id/approve` | Approve a knowledge candidate into the knowledge base. |
| POST | `/admin/knowledge/candidates/:id/reject` | Reject a knowledge candidate. |

## Compatibility Routes

| Method | Path | Description |
| --- | --- | --- |
| POST | `/v1/chat/completions` | OpenAI-compatible chat completions endpoint for Open-LLM-VTuber. |
| ANY | `/vtuber-ws/*path` | WebSocket reverse proxy to Open-LLM-VTuber on `127.0.0.1:12393`; authenticates via the HttpOnly `auth_token` Cookie or the `auth.token.<JWT>` subprotocol. URL query parameters are not accepted as a JWT authentication source. |
| POST | `/api/v1/track` | Lightweight page visit and user action tracking endpoint. Invalid payloads are ignored with a success-shaped response. |
| GET | `/health` | Service health check. |
