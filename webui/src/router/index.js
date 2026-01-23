import { createRouter, createWebHashHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import LoginView from '../views/LoginView.vue'
import ProfileView from '../views/ProfileView.vue'

const router = createRouter({
    history: createWebHashHistory(import.meta.env.BASE_URL),
    routes: [
        { path: '/', component: HomeView },
        { path: '/login', component: LoginView },
        { path: '/profile', component: ProfileView },
    ]
})

router.beforeEach((to, from, next) => {
    if (to.path !== '/login' && !localStorage.getItem('token')) {
        next('/login')
    } else {
        next()
    }
})

export default router
