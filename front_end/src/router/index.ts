// 路由与导航守卫：未登录跳转登录页，登录后回跳原地址
import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { public: true, title: '登录' },
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('@/views/RegisterView.vue'),
    meta: { public: true, title: '注册' },
  },
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    redirect: '/today',
    children: [
      {
        path: 'today',
        name: 'today',
        component: () => import('@/views/TodayView.vue'),
        meta: { title: '今日' },
      },
      {
        path: 'todos',
        name: 'todos',
        component: () => import('@/views/TodoListView.vue'),
        meta: { title: '全部待办' },
      },
      {
        path: 'calendar',
        name: 'calendar',
        component: () => import('@/views/CalendarView.vue'),
        meta: { title: '日历' },
      },
      {
        path: 'completed',
        name: 'completed',
        component: () => import('@/views/CompletedView.vue'),
        meta: { title: '已完成' },
      },
      {
        path: 'notes',
        name: 'notes',
        component: () => import('@/views/NotesView.vue'),
        meta: { title: '记事本' },
      },
      {
        path: 'habits',
        name: 'habits',
        component: () => import('@/views/HabitsView.vue'),
        meta: { title: '习惯打卡' },
      },
      {
        path: 'anniversaries',
        name: 'anniversaries',
        component: () => import('@/views/AnniversariesView.vue'),
        meta: { title: '纪念日' },
      },
      {
        path: 'notifications',
        name: 'notifications',
        component: () => import('@/views/NotificationsView.vue'),
        meta: { title: '提醒中心' },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/today',
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.ready) {
    await auth.bootstrap()
  }
  const publicPage = to.meta.public === true
  if (publicPage) {
    if (auth.authenticated) return { name: 'today' }
    return true
  }
  if (!auth.authenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  return true
})

export default router
