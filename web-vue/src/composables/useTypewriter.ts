import { ref, type Ref } from 'vue';

export interface UseTypewriterOptions {
  /** Characters per second (default 50) */
  speed?: number;
}

export interface UseTypewriterReturn {
  /** Current displayed text */
  displayText: Ref<string>;
  /** Whether typewriter is actively animating */
  isTyping: Ref<boolean>;
  /** Start typewriter effect for full text */
  startTypewriter: (fullText: string) => void;
  /** Immediately display full text (interrupt) */
  completeTypewriter: () => void;
  /** Clean up timer */
  dispose: () => void;
}

export function useTypewriter(options: UseTypewriterOptions = {}): UseTypewriterReturn {
  const { speed = 50 } = options;

  const displayText = ref('');
  const isTyping = ref(false);
  let timer: ReturnType<typeof setInterval> | null = null;
  let fullText = '';
  let charIndex = 0;

  const msPerChar = 1000 / speed;

  function startTypewriter(text: string) {
    // Stop any existing animation
    completeTypewriter();

    fullText = text;
    charIndex = 0;
    displayText.value = '';
    isTyping.value = true;

    if (!text) {
      isTyping.value = false;
      return;
    }

    timer = setInterval(() => {
      charIndex += 1;
      if (charIndex >= fullText.length) {
        displayText.value = fullText;
        isTyping.value = false;
        if (timer) clearInterval(timer);
        timer = null;
      } else {
        displayText.value = fullText.slice(0, charIndex);
      }
    }, msPerChar);
  }

  function completeTypewriter() {
    if (timer) {
      clearInterval(timer);
      timer = null;
    }
    if (fullText) {
      displayText.value = fullText;
    }
    isTyping.value = false;
  }

  function dispose() {
    completeTypewriter();
  }

  return {
    displayText,
    isTyping,
    startTypewriter,
    completeTypewriter,
    dispose,
  };
}
