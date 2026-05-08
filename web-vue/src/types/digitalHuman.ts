export type ConversationState = 'idle' | 'connecting' | 'listening' | 'thinking' | 'speaking' | 'interrupted' | 'error';

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  text: string;
  time: string;
}

export interface VtuberMessage {
  type: string;
  text?: string;
  audio?: string | null;
  volumes?: number[];
  slice_length?: number;
  display_text?: {
    text?: string;
    name?: string;
  };
  actions?: {
    expressions?: Array<string | number>;
    pictures?: string[];
    sounds?: string[];
  } | null;
  forwarded?: boolean;
  message?: string;
}
