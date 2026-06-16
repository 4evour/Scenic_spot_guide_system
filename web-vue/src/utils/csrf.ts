/** 从 cookie 中读取 csrf_token（Double-Submit Cookie 模式） */
export function getCSRFToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : '';
}
