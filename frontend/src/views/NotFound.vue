<template>
  <a-result status="404" title="404" sub-title="页面不存在或没有访问权限">
    <template #extra>
      <router-link to="/"><a-button type="primary">返回首页</a-button></router-link>
    </template>
  </a-result>
</template>
