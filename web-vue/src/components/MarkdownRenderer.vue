<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';

const props = withDefaults(defineProps<{
  text: string;
  /** Enable typewriter streaming effect */
  streaming?: boolean;
  /** Characters per second for typewriter (default 50) */
  speed?: number;
}>(), {
  streaming: false,
  speed: 50,
});

const rendered = ref('');
let timer: ReturnType<typeof setInterval> | null = null;
let charIndex = 0;
let fullText = '';

async function renderMarkdown(md: string): Promise<string> {
  if (!md) return '';
  try {
    const { marked } = await import('marked');
    const { default: DOMPurify } = await import('dompurify');

    const rawHtml = await marked.parse(md, { async: false }) as string;
    return DOMPurify.sanitize(rawHtml, {
      ALLOWED_TAGS: [
        'p', 'br', 'strong', 'em', 'u', 's', 'del',
        'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
        'ul', 'ol', 'li', 'blockquote', 'pre', 'code',
        'a', 'img', 'table', 'thead', 'tbody', 'tr', 'th', 'td',
      ],
      ALLOWED_ATTR: ['href', 'target', 'rel', 'src', 'alt', 'class'],
    });
  } catch {
    // Fallback: escape HTML and use as plain text
    return md.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  }
}

async function setFullText(text: string) {
  stopTypewriter();
  rendered.value = await renderMarkdown(text);
}

async function startTypewriter(text: string) {
  stopTypewriter();
  fullText = text;
  charIndex = 0;
  rendered.value = '';

  if (!text) return;

  const msPerChar = 1000 / props.speed;

  timer = setInterval(async () => {
    charIndex += 1;
    if (charIndex >= fullText.length) {
      rendered.value = await renderMarkdown(fullText);
      timer = null;
    } else {
      rendered.value = await renderMarkdown(fullText.slice(0, charIndex));
    }
  }, msPerChar);
}

function stopTypewriter() {
  if (timer) {
    clearInterval(timer);
    timer = null;
  }
}

async function completeTypewriter() {
  stopTypewriter();
  if (fullText) {
    rendered.value = await renderMarkdown(fullText);
  }
}

watch(() => props.text, async (newText) => {
  if (props.streaming) {
    await startTypewriter(newText);
  } else {
    await setFullText(newText);
  }
}, { immediate: true });

onMounted(() => {
  // Initial render
  if (!props.streaming) {
    setFullText(props.text);
  }
});
</script>

<template>
  <div class="markdown-renderer" v-html="rendered" />
</template>

<style scoped>
.markdown-renderer :deep(p) {
  margin: 0 0 6px;
}
.markdown-renderer :deep(p:last-child) {
  margin-bottom: 0;
}
.markdown-renderer :deep(strong) {
  color: var(--sg-text-body, rgba(255, 255, 255, 0.9));
  font-weight: 600;
}
.markdown-renderer :deep(em) {
  color: var(--sg-jade-bright, #63e2b7);
}
.markdown-renderer :deep(a) {
  color: var(--sg-cyan, #52f0ee);
  text-decoration: underline;
}
.markdown-renderer :deep(a:hover) {
  color: var(--sg-jade-bright, #63e2b7);
}
.markdown-renderer :deep(ul),
.markdown-renderer :deep(ol) {
  margin: 4px 0;
  padding-left: 20px;
}
.markdown-renderer :deep(li) {
  margin: 2px 0;
  line-height: 1.6;
}
.markdown-renderer :deep(code) {
  background: rgba(255, 255, 255, 0.08);
  padding: 1px 5px;
  border-radius: 3px;
  font-size: 0.9em;
  font-family: 'Fira Code', 'Consolas', monospace;
  color: var(--sg-gold, #f4c765);
}
.markdown-renderer :deep(pre) {
  background: rgba(0, 0, 0, 0.3);
  padding: 10px 14px;
  border-radius: 6px;
  overflow-x: auto;
  margin: 6px 0;
}
.markdown-renderer :deep(pre code) {
  background: none;
  padding: 0;
  color: var(--sg-text-secondary, rgba(255, 255, 255, 0.75));
}
.markdown-renderer :deep(blockquote) {
  border-left: 3px solid var(--sg-jade-border, rgba(99, 226, 183, 0.3));
  margin: 6px 0;
  padding: 4px 12px;
  color: var(--sg-text-hint, rgba(255, 255, 255, 0.5));
  background: rgba(99, 226, 183, 0.04);
  border-radius: 0 4px 4px 0;
}
.markdown-renderer :deep(h1),
.markdown-renderer :deep(h2),
.markdown-renderer :deep(h3) {
  color: var(--sg-text-body, rgba(255, 255, 255, 0.88));
  margin: 8px 0 4px;
  font-weight: 600;
}
.markdown-renderer :deep(h1) { font-size: 1.2em; }
.markdown-renderer :deep(h2) { font-size: 1.1em; }
.markdown-renderer :deep(h3) { font-size: 1em; }
.markdown-renderer :deep(table) {
  border-collapse: collapse;
  margin: 6px 0;
  width: 100%;
}
.markdown-renderer :deep(th),
.markdown-renderer :deep(td) {
  border: 1px solid rgba(255, 255, 255, 0.1);
  padding: 6px 10px;
  text-align: left;
  font-size: 0.9em;
}
.markdown-renderer :deep(th) {
  background: rgba(255, 255, 255, 0.06);
}
</style>
