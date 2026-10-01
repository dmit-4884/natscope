import DefaultTheme from 'vitepress/theme'
import Home from './components/Home.vue'
import './custom.css'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    app.component('Home', Home)
  }
}
