export type PlaybackHooks = {
  onStart?: (text?: string, cue?: PlaybackCue) => void;
  onEnd?: () => void;
  onVolume?: (volume: number) => void;
};

export type PlaybackCue = {
  volumes?: number[];
  sliceLengthMs?: number;
  expression?: string;
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

  async playTextFallback(text: string, cue: PlaybackCue = {}) {
    if (!('speechSynthesis' in window)) return;
    if (this.interrupted) return;
    const token = this.playbackToken;
    window.speechSynthesis.cancel();
    const utterance = new SpeechSynthesisUtterance(text);
    utterance.lang = 'zh-CN';
    utterance.rate = 0.95;
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
    };
    window.speechSynthesis.speak(utterance);
  }

  resume() {
    this.interrupted = false;
    this.playbackToken += 1;
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
    } catch {
      if (this.interrupted || token !== this.playbackToken) return;
      this.stopVolumeTracking();
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
