<template>
  <div class="thinking-indicator">
    <span class="think-dot" style="animation-delay: 0s" />
    <span class="think-dot" style="animation-delay: 0.2s" />
    <span class="think-dot" style="animation-delay: 0.4s" />
  </div>
</template>

<style scoped>
.thinking-indicator {
  position: absolute;
  top: 16%;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  gap: 6px;
  z-index: 3;
}
.think-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--sg-cyan, #52f0ee), var(--sg-jade-bright, #63e2b7));
  animation: dot-bounce 1.2s ease-in-out infinite;
}
@keyframes dot-bounce {
  0%, 80%, 100% { transform: translateY(0); opacity: 0.4; }
  40% { transform: translateY(-10px); opacity: 1; }
}
</style>
