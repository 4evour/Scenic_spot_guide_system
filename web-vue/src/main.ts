import { createApp } from 'vue';
import App from './App.vue';
import { startHashRouter } from './router';
import './styles/global.css';

startHashRouter();
createApp(App).mount('#app');
