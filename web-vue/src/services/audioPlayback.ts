import { getBrowserSpeechRate } from '../composables/useSeniorMode';
import i18n from '../i18n';

export type PlaybackHooks = {
  onStart?: (text?: string, cue?: PlaybackCue) => void;
  onEnd?: () => void;
  onComplete?: () => void;
  onFirstByte?: () => void;
  onVolume?: (volume: number) => void;
  onError?: (message: string) => void;
};

export type PlaybackCue = {
  volumes?: number[];
  sliceLengthMs?: number;
  expression?: string;
  showText?: boolean;
};

type QueueItem = {
  url: string;
  text?: string;
  revokeAfterUse?: boolean;
} & PlaybackCue;

export class AudioPlaybackController {
  private audio: HTMLAudioElement | null = null;
  private queue: QueueItem[] = [];
  private interrupted = false;
  private playbackToken = 0;
  private hooks: PlaybackHooks;
  private audioContext: AudioContext | null = null;
  private analyser: AnalyserNode | null = null;
  private mediaSource: MediaElementAudioSourceNode | null = null;
  private analyserBuffer: Uint8Array<ArrayBuffer> | null = null;
  private volumeRaf = 0;
  private speechPulseTimer = 0;
  private smoothedVolume = 0;
  private masterVolume = 1;

  constructor(hooks: PlaybackHooks = {}) {
    this.hooks = hooks;
  }

  setVolume(volume: number) {
    this.masterVolume = this.clampVolume(volume);
    if (this.audio) {
      this.audio.volume = this.masterVolume;
    }
  }

  enqueueBase64Wav(audioBase64: string, text?: string, cue: PlaybackCue = {}) {
    if (this.interrupted) return false;
    let url: string;
    try {
      const binary = window.atob(audioBase64);
      const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0));
      url = URL.createObjectURL(new Blob([bytes], { type: 'audio/wav' }));
    } catch {
      return false;
    }
    this.queue.push({ url, text, revokeAfterUse: true, ...cue });
    this.hooks.onFirstByte?.();
    if (!this.audio) void this.playNext();
    return true;
  }

  enqueueUrl(url: string, text?: string, cue: PlaybackCue = {}) {
    if (this.interrupted) return false;
    this.queue.push({ url, text, ...cue });
    this.hooks.onFirstByte?.();
    if (!this.audio) void this.playNext();
    return true;
  }

  /**
   * 流式播放：从 fetch Response 的 ReadableStream 渐进消费音频分片，
   * 边下载边播放，降低首音等待时间。
   * 使用 MediaSource API 实现。
   */
  async enqueueStream(fetchResponse: Response, text?: string, cue: PlaybackCue = {}): Promise<boolean> {
    if (this.interrupted) return false;
    const token = this.playbackToken;

    try {
      const responseBody = fetchResponse.body;
      if (!responseBody || !('MediaSource' in window) || !MediaSource.isTypeSupported('audio/mpeg')) {
        return false;
      }
      const mediaSource = new MediaSource();
      const url = URL.createObjectURL(mediaSource);
      let appendedBytes = 0;

      const queueItem: QueueItem = { url, text, revokeAfterUse: true, ...cue };
      // Push to queue — the standard playNext will handle it when it's this item's turn.
      this.queue.push(queueItem);

      // If nothing is playing, kick off playback now.
      const shouldStart = !this.audio;
      if (shouldStart) {
        void this.playNext();
      }

      // Wait for the MediaSource to open, then feed chunks.
      mediaSource.addEventListener('sourceopen', async () => {
        if (this.interrupted || token !== this.playbackToken) {
          mediaSource.endOfStream();
          return;
        }

        const sourceBuffer = mediaSource.addSourceBuffer('audio/mpeg');
        const reader = responseBody.getReader();

        try {
          while (true) {
            if (this.interrupted || token !== this.playbackToken) {
              reader.cancel();
              break;
            }

            const { done, value } = await reader.read();
            if (done) break;
            if (value.byteLength > 0 && appendedBytes === 0) this.hooks.onFirstByte?.();
            appendedBytes += value.byteLength;

            // Wait for any pending append to finish
            await waitForSourceBufferUpdate(sourceBuffer);
            if (this.interrupted || token !== this.playbackToken) {
              reader.cancel();
              break;
            }

            sourceBuffer.appendBuffer(value);
          }
        } catch {
          // Stream aborted or error — end gracefully
        } finally {
          await waitForSourceBufferUpdate(sourceBuffer);
          if (mediaSource.readyState === 'open') {
            try { mediaSource.endOfStream(); } catch { /* ignore */ }
          }
          if (appendedBytes === 0 && !this.interrupted && token === this.playbackToken) {
            this.hooks.onError?.(i18n.global.t('dh.audio.emptyStreamFallback'));
            void this.playTextFallback(text || '', cue);
          }
        }
      });

      return true;
    } catch {
      // MediaSource 不支持时降级到 URL 播放
      return false;
    }
  }

  async playTextFallback(text: string, cue: PlaybackCue = {}): Promise<boolean> {
    const SpeechSynthesisUtteranceCtor = (window as Window & {
      SpeechSynthesisUtterance?: typeof SpeechSynthesisUtterance;
    }).SpeechSynthesisUtterance;
    if (!('speechSynthesis' in window) || !SpeechSynthesisUtteranceCtor) return false;
    if (this.interrupted) return false;
    if (!text.trim()) return false;
    const token = this.playbackToken;
    window.speechSynthesis.cancel();
    const utterance = new SpeechSynthesisUtteranceCtor(text);
    utterance.lang = i18n.global.locale.value;
    utterance.rate = getBrowserSpeechRate();
    utterance.pitch = 1.05;
    utterance.volume = this.masterVolume;
    utterance.addEventListener('start', () => {
      if (this.interrupted || token !== this.playbackToken) return;
      this.hooks.onStart?.(text, cue);
      this.startSpeechPulse(token);
    });
    utterance.addEventListener('end', () => {
      if (token !== this.playbackToken) return;
      this.stopVolumeTracking();
      this.hooks.onEnd?.();
      this.hooks.onComplete?.();
    });
    utterance.addEventListener('error', () => {
      if (token !== this.playbackToken) return;
      this.stopVolumeTracking();
      this.hooks.onEnd?.();
      this.hooks.onError?.(i18n.global.t('dh.audio.browserSpeechBlocked'));
    });
    window.speechSynthesis.speak(utterance);
    return true;
  }

  resume() {
    this.interrupted = false;
    this.playbackToken += 1;
    if (this.audioContext?.state === 'suspended') {
      void this.audioContext.resume();
    }
  }

  async unlock(): Promise<boolean> {
    this.interrupted = false;
    try {
      const AudioContextCtor = window.AudioContext || (window as Window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
      if (AudioContextCtor) {
        this.audioContext ??= new AudioContextCtor();
        if (this.audioContext.state === 'suspended') await this.audioContext.resume();
      }
      const SpeechSynthesisUtteranceCtor = (window as Window & {
        SpeechSynthesisUtterance?: typeof SpeechSynthesisUtterance;
      }).SpeechSynthesisUtterance;
      if ('speechSynthesis' in window && SpeechSynthesisUtteranceCtor) {
        window.speechSynthesis.cancel();
        const utterance = new SpeechSynthesisUtteranceCtor(' ');
        utterance.volume = 0;
        window.speechSynthesis.speak(utterance);
        window.speechSynthesis.cancel();
      }
      return true;
    } catch {
      this.hooks.onError?.(i18n.global.t('dh.audio.soundPermissionDenied'));
      return false;
    }
  }

  interrupt() {
    this.interrupted = true;
    this.playbackToken += 1;
    for (const item of this.queue) this.revokeItemUrl(item);
    this.queue = [];
    this.stopVolumeTracking();
    if (this.audio) {
      this.audio.pause();
      if (this.audio.src.startsWith('blob:')) URL.revokeObjectURL(this.audio.src);
      this.audio.src = '';
      this.audio.load();
      this.audio = null;
    }
    if ('speechSynthesis' in window) window.speechSynthesis.cancel();
    this.hooks.onEnd?.();
  }

  private async playNext(completeWhenEmpty = true) {
    if (this.interrupted) return;
    const token = this.playbackToken;
    const item = this.queue.shift();
    if (!item) {
      this.audio = null;
      this.hooks.onEnd?.();
      if (completeWhenEmpty) this.hooks.onComplete?.();
      return;
    }

    const audio = new Audio(item.url);
    this.audio = audio;
    audio.volume = this.masterVolume;
    audio.addEventListener('play', () => {
      if (this.interrupted || token !== this.playbackToken) return;
      this.hooks.onStart?.(item.text, item);
      this.startVolumeTracking(audio, item, token);
    });
    audio.addEventListener('ended', () => {
      if (this.interrupted || token !== this.playbackToken) return;
      this.stopVolumeTracking();
      this.revokeItemUrl(item);
      this.audio = null;
      void this.playNext();
    });
    audio.addEventListener('error', () => {
      if (this.interrupted || token !== this.playbackToken) return;
      this.stopVolumeTracking();
      this.revokeItemUrl(item);
      this.audio = null;
      void this.playNext(false);
    });

    try {
      await audio.play();
    } catch (err) {
      if (this.interrupted || token !== this.playbackToken) return;
      this.stopVolumeTracking();
      this.hooks.onError?.(isAutoplayError(err) ? i18n.global.t('dh.audio.autoplayBlocked') : i18n.global.t('dh.audio.playbackFailedNext'));
      this.revokeItemUrl(item);
      this.audio = null;
      void this.playNext(false);
    }
  }

  private revokeItemUrl(item: QueueItem) {
    if (item.revokeAfterUse) URL.revokeObjectURL(item.url);
  }

  private startVolumeTracking(audio: HTMLAudioElement, item: QueueItem, token: number) {
    this.stopVolumeTracking(false);
    this.smoothedVolume = 0;
    const hasVolumeFrames = Boolean(item.volumes?.length && item.sliceLengthMs);
    const analyser = hasVolumeFrames ? null : this.createAnalyser(audio);

    const tick = () => {
      if (this.interrupted || token !== this.playbackToken || audio !== this.audio) return;
      const target = hasVolumeFrames ? this.volumeFromFrames(audio, item) : this.volumeFromAnalyser(analyser, audio);
      this.smoothedVolume = this.smoothedVolume * 0.35 + target * 0.65;
      this.hooks.onVolume?.(this.clampVolume(this.smoothedVolume));
      this.volumeRaf = window.requestAnimationFrame(tick);
    };

    tick();
  }

  private startSpeechPulse(token: number) {
    this.stopVolumeTracking(false);
    this.speechPulseTimer = window.setInterval(() => {
      if (this.interrupted || token !== this.playbackToken) return;
      const t = performance.now() / 1000;
      const value = 0.2 + Math.max(0, Math.sin(t * 18)) * 0.48 + Math.max(0, Math.sin(t * 31)) * 0.18;
      this.hooks.onVolume?.(this.clampVolume(value));
    }, 45);
  }

  private stopVolumeTracking(emitZero = true) {
    window.cancelAnimationFrame(this.volumeRaf);
    this.volumeRaf = 0;
    window.clearInterval(this.speechPulseTimer);
    this.speechPulseTimer = 0;
    this.smoothedVolume = 0;
    this.disconnectAnalyser();
    if (emitZero) this.hooks.onVolume?.(0);
  }

  private volumeFromFrames(audio: HTMLAudioElement, item: QueueItem) {
    if (!item.volumes?.length || !item.sliceLengthMs) return 0;
    const index = Math.min(item.volumes.length - 1, Math.max(0, Math.floor((audio.currentTime * 1000) / item.sliceLengthMs)));
    const volume = item.volumes[index] ?? 0;
    return this.clampVolume(volume);
  }

  private createAnalyser(audio: HTMLAudioElement) {
    try {
      const AudioContextCtor = window.AudioContext || (window as Window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
      if (!AudioContextCtor) return null;
      this.audioContext ??= new AudioContextCtor();
      void this.audioContext.resume();
      this.analyser = this.audioContext.createAnalyser();
      this.analyser.fftSize = 512;
      this.analyser.smoothingTimeConstant = 0.35;
      this.analyserBuffer = new Uint8Array(new ArrayBuffer(this.analyser.fftSize));
      this.mediaSource = this.audioContext.createMediaElementSource(audio);
      this.mediaSource.connect(this.analyser);
      this.analyser.connect(this.audioContext.destination);
      return this.analyser;
    } catch {
      return null;
    }
  }

  private volumeFromAnalyser(analyser: AnalyserNode | null, audio: HTMLAudioElement) {
    if (!analyser || !this.analyserBuffer) {
      if (audio.paused || audio.ended) return 0;
      const t = audio.currentTime;
      return this.clampVolume(0.18 + Math.max(0, Math.sin(t * 20)) * 0.5);
    }

    analyser.getByteTimeDomainData(this.analyserBuffer);
    let sum = 0;
    for (const sample of this.analyserBuffer) {
      const centered = (sample - 128) / 128;
      sum += centered * centered;
    }
    const rms = Math.sqrt(sum / this.analyserBuffer.length);
    return this.clampVolume(rms * 3.2);
  }

  private disconnectAnalyser() {
    try {
      this.mediaSource?.disconnect();
      this.analyser?.disconnect();
    } catch {
      // Some browsers throw when disconnecting a node that is already detached.
    }
    this.mediaSource = null;
    this.analyser = null;
    this.analyserBuffer = null;
  }

  private clampVolume(volume: number) {
    if (!Number.isFinite(volume)) return 0;
    return Math.min(1, Math.max(0, volume));
  }
}

function isAutoplayError(err: unknown) {
  return err instanceof DOMException && (err.name === 'NotAllowedError' || err.name === 'NotSupportedError');
}

/**
 * 等待 SourceBuffer 完成上一次 append/remove 操作。
 * SourceBuffer 不能并发 append，必须等 updateend 事件。
 */
function waitForSourceBufferUpdate(buffer: SourceBuffer): Promise<void> {
  return new Promise((resolve) => {
    if (!buffer.updating) {
      resolve();
      return;
    }
    const onEnd = () => {
      buffer.removeEventListener('updateend', onEnd);
      resolve();
    };
    buffer.addEventListener('updateend', onEnd);
  });
}
