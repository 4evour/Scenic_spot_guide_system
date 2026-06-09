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

## Scenic Spots

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/spots` | Public | List scenic spots. |
| GET | `/spots/:id` | Public | Get one scenic spot. |
| GET | `/spots/category` | Public | List spots by category. |
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
| POST | `/routes` | Admin | Create a route. |
| PUT | `/routes/:id` | Admin | Update a route. |
| DELETE | `/routes/:id` | Admin | Delete a route. |

## Visitor Queries

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
| POST | `/ai/chat` | Public | Ask the RAG chat service. |
| POST | `/ai/tts` | Public | Build a Baidu TTS audio URL. |
| GET | `/knowledge/list` | Admin | List knowledge chunks with database pagination and filters. |
| GET | `/knowledge/:id` | Admin | Get one knowledge chunk. |
| POST | `/knowledge` | Admin | Create a knowledge chunk. |
| POST | `/knowledge/upload` | Admin | Upload JSONL, JSON, Markdown, or TXT, max 10 MB. |
| PUT | `/knowledge/:id` | Admin | Update a knowledge chunk. |
| DELETE | `/knowledge/:id` | Admin | Delete a knowledge chunk. |
| DELETE | `/knowledge/all?confirm=DELETE_ALL_KNOWLEDGE` | Admin | Delete all knowledge chunks. |

## Digital Human

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
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
| GET | `/admin/reports/visitor` | Visitor report. |
| GET | `/admin/digital-human/config` | Digital human configuration. |
| PUT | `/admin/digital-human/config` | Update digital human configuration. |
| GET | `/admin/settings` | System settings. |
| PUT | `/admin/settings` | Update system settings. |
| GET | `/admin/knowledge/stats` | Knowledge base statistics. |

## Compatibility Routes

| Method | Path | Description |
| --- | --- | --- |
| POST | `/v1/chat/completions` | OpenAI-compatible chat completions endpoint for Open-LLM-VTuber. |
| ANY | `/vtuber-ws/*path` | WebSocket reverse proxy to Open-LLM-VTuber on `127.0.0.1:12393`; authenticates via `auth_token` Cookie, `auth.token.*` subprotocol, or legacy `?token=`. |
| GET | `/health` | Service health check. |
