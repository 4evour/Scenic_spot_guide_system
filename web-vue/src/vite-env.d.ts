/// <reference types="vite/client" />

declare global {
  type SpeechRecognitionConstructor = new () => SpeechRecognition;

  interface SpeechRecognition extends EventTarget {
    lang: string;
    interimResults: boolean;
    continuous: boolean;
    onstart: (() => void) | null;
    onend: (() => void) | null;
    onerror: ((event: SpeechRecognitionErrorEvent) => void) | null;
    onresult: ((event: SpeechRecognitionEvent) => void) | null;
    start(): void;
    stop(): void;
  }

  interface SpeechRecognitionEvent extends Event {
    results: SpeechRecognitionResultList;
  }

  interface SpeechRecognitionErrorEvent extends Event {
    error:
      | 'aborted'
      | 'audio-capture'
      | 'bad-grammar'
      | 'language-not-supported'
      | 'network'
      | 'no-speech'
      | 'not-allowed'
      | 'phrases-not-supported'
      | 'service-not-allowed';
    message: string;
  }

  interface SpeechRecognitionAlternative {
    transcript: string;
    confidence: number;
  }

  interface SpeechRecognitionResult {
    readonly length: number;
    readonly isFinal: boolean;
    item(index: number): SpeechRecognitionAlternative;
    [index: number]: SpeechRecognitionAlternative;
  }

  interface SpeechRecognitionResultList {
    readonly length: number;
    item(index: number): SpeechRecognitionResult;
    [index: number]: SpeechRecognitionResult;
  }

  interface Window {
    SpeechRecognition?: SpeechRecognitionConstructor;
    webkitSpeechRecognition?: SpeechRecognitionConstructor;
    PIXI?: unknown;
    Live2D?: unknown;
    Live2DModelWebGL?: unknown;
  }
}

declare module 'pixi.js';
declare module 'pixi-live2d-display/cubism4';

export {};
