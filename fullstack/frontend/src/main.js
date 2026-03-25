import { createApp } from 'vue'
import 'vue-sonner/style.css'

import App from './App.vue'
import router from './router'
import './styles.css'

const app = createApp(App)

app.use(router)
app.mount('#app')
