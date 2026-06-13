import { apiFetch } from './api';

export interface TTSOptions {
  text: string;
  voice?: string;
  rate?: string;
}

/**
 * 调用流式 TTS，返回 ReadableStream 用于渐进式播放。
 * 后端以 chunked transfer 逐块返回 audio/mpeg 数据。
 */
export async function streamTTS(options: TTSOptions): Promise<Response> {
  const csrfMatch = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/);
  const csrfToken = csrfMatch ? decodeURIComponent(csrfMatch[1]) : '';

  const response = await fetch('/api/v1/ai/tts/stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(csrfToken ? { 'X-CSRF-Token': csrfToken } : {}),
    },
    body: JSON.stringify({
      text: options.text,
      voice: options.voice || 'female_xiaoxiao',
      rate: options.rate || '+0%',
    }),
    credentials: 'include',
  });

  if (!response.ok) {
    const errText = await response.text().catch(() => '');
    throw new Error(errText || `TTS 请求失败 (${response.status})`);
  }

  return response;
}

/**
 * 非流式 TTS：等待完整音频后返回 Blob URL。
 */
export async function synthesizeTTS(options: TTSOptions): Promise<string> {
  const csrfMatch = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/);
  const csrfToken = csrfMatch ? decodeURIComponent(csrfMatch[1]) : '';

  const response = await fetch('/api/v1/ai/tts', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(csrfToken ? { 'X-CSRF-Token': csrfToken } : {}),
    },
    body: JSON.stringify({
      text: options.text,
      voice: options.voice || 'female_xiaoxiao',
      rate: options.rate || '+0%',
    }),
    credentials: 'include',
  });

  if (!response.ok) {
    const errText = await response.text().catch(() => '');
    throw new Error(errText || `TTS 请求失败 (${response.status})`);
  }

  const blob = await response.blob();
  return URL.createObjectURL(blob);
}
