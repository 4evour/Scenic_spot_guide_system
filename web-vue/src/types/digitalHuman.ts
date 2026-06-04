export type ConversationState = 'idle' | 'connecting' | 'listening' | 'thinking' | 'speaking' | 'interrupted' | 'error';

/** Minimal interface for PixiJS Application used by Live2D */
export interface PixiAppLike {
  view: HTMLCanvasElement;
  stage: { addChild(child: unknown): void };
  renderer: { resize(w: number, h: number): void };
  ticker: { add(fn: (dt: number) => void): void; remove(fn: (dt: number) => void): void; deltaMS: number };
  destroy(removeView?: boolean): void;
  [key: string]: unknown;
}

/** Minimal interface for Live2D model (pixi-live2d-display) */
export interface Live2DModelLike {
  anchor: { set(x: number, y: number): void };
  scale: { set(...args: number[]): void };
  x: number;
  y: number;
  autoUpdate: boolean;
  motion?(group: string): unknown;
  expression(expr: string): Promise<boolean>;
  update?(dt: number): void;
  getLocalBounds?(): { width: number; height: number };
  internalModel?: Live2DInternalModel;
  [key: string]: unknown;
}

interface Live2DInternalModel {
  coreModel?: {
    setParameterValueById(id: string, value: number, weight?: number): void;
    [key: string]: unknown;
  };
  motionManager?: {
    expressionManager?: { setExpression(expr: string): void };
    [key: string]: unknown;
  };
  on?(event: string, fn: unknown): void;
  off?(event: string, fn?: unknown): void;
  [key: string]: unknown;
}

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
