<script setup lang="ts">
import { computed, reactive } from 'vue';
import KpiCard from '../components/KpiCard.vue';
import BarList from '../components/BarList.vue';

const state = reactive({
  tab: 'knowledge',
  search: '',
  knowledge: [
    { title: '灵山主峰历史讲解词', category: '历史文化', content: '主峰海拔 886 米，沿线串联古道遗存、观景平台和地方民俗故事，是文化型游客的核心讲解点。', updated: '2026-05-08' },
    { title: '游客服务设施 FAQ', category: '服务设施', content: '停车场、卫生间、游客中心、母婴室和医疗点均可通过数字人问答快速定位。', updated: '2026-05-07' },
    { title: '亲子路线推荐模板', category: '路线推荐', content: '优先推荐低坡度路线、互动打卡点和文创驿站，控制游览时长在 60 到 90 分钟。', updated: '2026-05-06' },
    { title: '开放时间与票务说明', category: '游客 FAQ', content: '开放时间为 08:00-17:30，节假日可根据客流热力动态推荐入园时间。', updated: '2026-05-05' },
  ],
});

const tabs = [
  ['knowledge', '知识库管理'],
  ['avatar', '数字人形象'],
  ['reports', '感受度报告'],
  ['settings', '系统设置'],
];

const filtered = computed(() => {
  const term = state.search.trim().toLowerCase();
  if (!term) return state.knowledge;
  return state.knowledge.filter(item => `${item.title}${item.category}${item.content}`.toLowerCase().includes(term));
});

function addMockKnowledge() {
  state.knowledge.unshift({
    title: '新增演示知识条目',
    category: '游客 FAQ',
    content: '这是一条虚拟知识内容，用于演示后台上传、维护和检索能力。',
    updated: new Date().toLocaleDateString('zh-CN'),
  });
}
</script>

<template>
  <main class="admin-view">
    <aside class="side-console">
      <div class="brand-block">
        <span>AI</span>
        <div>
          <strong>灵山智慧导览</strong>
          <small>Management Console</small>
        </div>
      </div>
      <button v-for="[key, label] in tabs" :key="key" :class="{ active: state.tab === key }" @click="state.tab = key">
        {{ label }}
      </button>
    </aside>

    <section class="admin-content">
      <header class="hero-console compact">
        <div>
          <p class="eyebrow">Admin Console</p>
          <h1>{{ tabs.find(([key]) => key === state.tab)?.[1] }}</h1>
          <p>覆盖知识库、数字人形象、游客感受度报告和系统配置，数据采用演示模拟源。</p>
        </div>
      </header>

      <section class="kpi-grid four">
        <KpiCard label="知识条目" :value="String(1286 + state.knowledge.length)" note="本周新增 +128" />
        <KpiCard label="问答准确率" value="93.6%" note="标准测试集模拟" tone="green" />
        <KpiCard label="平均延迟" value="2.1s" note="语音问答链路" tone="gold" />
        <KpiCard label="满意度" value="96.8%" note="交互反馈统计" />
      </section>

      <section v-if="state.tab === 'knowledge'" class="panel">
        <div class="toolbar">
          <input v-model="state.search" placeholder="搜索景点、讲解词、FAQ 或路线..." />
          <button class="primary-action" @click="addMockKnowledge">新增知识</button>
        </div>
        <div class="knowledge-grid">
          <article v-for="item in filtered" :key="item.title" class="knowledge-card-vue">
            <div>
              <h3>{{ item.title }}</h3>
              <span>{{ item.category }}</span>
            </div>
            <p>{{ item.content }}</p>
            <small>更新时间：{{ item.updated }}</small>
          </article>
        </div>
      </section>

      <section v-if="state.tab === 'avatar'" class="admin-two">
        <article class="panel avatar-config-preview">
          <div class="avatar-holo">小灵</div>
          <h2>数字人形象配置</h2>
          <p>当前形象使用 Live2D 模型，支持语音、口型、表情和动作状态联动。</p>
        </article>
        <article class="panel form-panel">
          <label>数字人名称<input value="小灵" /></label>
          <label>讲解声音<select><option>温柔自然女声</option><option>沉稳专业女声</option></select></label>
          <label>服装风格<select><option>景区文旅制服</option><option>山水讲解员制服</option></select></label>
          <label>语速<input type="range" min="0.7" max="1.3" step="0.1" value="0.9" /></label>
          <button class="primary-action">保存配置</button>
        </article>
      </section>

      <section v-if="state.tab === 'reports'" class="admin-two">
        <article class="panel span-2">
          <h2>游客关注点分析</h2>
          <BarList :items="[{label:'开放时间', value:92},{label:'主峰路线', value:88},{label:'历史文化', value:81},{label:'亲子服务', value:74},{label:'雨天方案', value:56}]" />
        </article>
        <article class="panel">
          <h2>服务建议</h2>
          <ul class="clean-list">
            <li>上午 10 点后路线咨询明显增加，建议首页展示快捷路线推荐。</li>
            <li>老人和亲子问题占比上升，可补充无障碍与亲子服务知识。</li>
            <li>雨天场景需要加入室内展陈和避雨点推荐。</li>
          </ul>
        </article>
      </section>

      <section v-if="state.tab === 'settings'" class="panel form-panel">
        <label>景区名称<input value="灵山景区" /></label>
        <label>服务热线<input value="400-168-0303" /></label>
        <label>系统提示词<textarea rows="5">你是灵山景区 AI 导游，请基于本地知识库回答游客问题，优先保证事实准确、语气自然、路线建议可执行。</textarea></label>
        <label class="toggle-row"><input type="checkbox" checked /> 启用知识库检索增强</label>
        <label class="toggle-row"><input type="checkbox" checked /> 启用游客感受度分析</label>
      </section>
    </section>
  </main>
</template>
