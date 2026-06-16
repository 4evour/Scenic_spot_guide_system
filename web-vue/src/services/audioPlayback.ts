import { getBrowserSpeechRate } from '../composables/useSeniorMode';

export type PlaybackHooks = {
  onStart?: (text?: string, cue?: PlaybackCue) => void;
  onEnd?: () => void;
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

  constructor(hooks: PlaybackHooks = {}) {
    this.hooks = hooks;
  }

  enqueueBase64Wav(audioBase64: string, text?: string, cue: PlaybackCue = {}) {
    if (this.interrupted) return false;
    const url = `data:audio/wav;base64,${audioBase64}`;
    this.queue.push({ url, text, ...cue });
    if (!this.audio) void this.playNext();
    return true;
  }

  enqueueUrl(url: string, text?: string, cue: PlaybackCue = {}) {
    if (this.interrupted) return false;
    this.queue.push({ url, text, ...cue });
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

      const queueItem: QueueItem = { url, text, ...cue };
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
            this.hooks.onError?.('语音合成没有返回音频，已切换为浏览器朗读。');
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

  async playTextFallback(text: string, cue: PlaybackCue = {}) {
    if (!('speechSynthesis' in window)) return;
    if (this.interrupted) return;
    if (!text.trim()) return;
    const token = this.playbackToken;
    window.speechSynthesis.cancel();
    const utterance = new SpeechSynthesisUtterance(text);
    utterance.lang = 'zh-CN';
    utterance.rate = getBrowserSpeechRate();
    utterance.pitch = 1.05;
    utterance.onstart = () => {
      if (this.interrupted || token !== this.playbackToken) return;
      this.hooks.onStart?.(text, cue);
      this.startSpeechPulse(token);
    };
    utterance.onend = () => {
      if (token !== this.playbackToken) return;
      this.stopVolumeTracking();
      this.hooks.onEnd?.();
    };
    utterance.onerror = () => {
      if (token !== this.playbackToken) return;
      this.stopVolumeTracking();
      this.hooks.onEnd?.();
      this.hooks.onError?.('浏览器朗读被阻止，请点击“启用声音”后重试。');
    };
    window.speechSynthesis.speak(utterance);
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
      if ('speechSynthesis' in window) {
        window.speechSynthesis.cancel();
        const utterance = new SpeechSynthesisUtterance(' ');
        utterance.volume = 0;
        window.speechSynthesis.speak(utterance);
        window.speechSynthesis.cancel();
      }
      return true;
    } catch {
      this.hooks.onError?.('浏览器未允许播放声音，请检查站点声音权限。');
      return false;
    }
  }

  interrupt() {
    this.interrupted = true;
    this.playbackToken += 1;
    this.queue = [];
    this.stopVolumeTracking();
    if (this.audio) {
      this.audio.pause();
      this.audio.src = '';
      this.audio.load();
      this.audio = null;
    }
    if ('speechSynthesis' in window) window.speechSynthesis.cancel();
    this.hooks.onEnd?.();
  }

  private async playNext() {
    if (this.interrupted) return;
    const token = this.playbackToken;
    const item = this.queue.shift();
    if (!item) {
      this.audio = null;
      this.hooks.onEnd?.();
      return;
    }

    this.audio = new Audio(item.url);
    this.audio.onplay = () => {
      if (this.interrupted || token !== this.playbackToken) return;
      this.hooks.onStart?.(item.text, item);
      this.startVolumeTracking(this.audio!, item, token);
    };
    this.audio.onended = () => {
      if (this.interrupted || token !== this.playbackToken) return;
      this.stopVolumeTracking();
      this.audio = null;
      void this.playNext();
    };
    this.audio.onerror = () => {
      if (this.interrupted || token !== this.playbackToken) return;
      this.stopVolumeTracking();
      this.audio = null;
      void this.playNext();
    };

    try {
      await this.audio.play();
    } catch (err) {
      if (this.interrupted || token !== this.playbackToken) return;
      this.stopVolumeTracking();
      this.hooks.onError?.(isAutoplayError(err) ? '浏览器阻止了自动播放，请点击“启用声音”。' : '音频播放失败，已尝试播放下一段。');
      this.audio = null;
      void this.playNext();
    }
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
