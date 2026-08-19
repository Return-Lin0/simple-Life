<template>
  <!-- 页面级切换动画：丝滑连贯 -->
  <router-view v-slot="{ Component }">
    <transition name="page" mode="out-in">
      <component :is="Component" />
    </transition>
  </router-view>
</template>

<script setup lang="ts">
// 根组件：仅承载路由出口与全局过渡动画
</script>
