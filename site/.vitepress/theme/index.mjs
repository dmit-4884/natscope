import DefaultTheme from 'vitepress/theme'
import Video from './components/Video.vue'
import Shot from './components/Shot.vue'
import './custom.css'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    app.component('Video', Video)
    app.component('Shot', Shot)
  }
}
