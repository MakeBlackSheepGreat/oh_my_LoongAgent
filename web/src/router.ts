// 极简 hash 路由（npm 受限，spec task8.0 允许手写）：login / workbench 两个视图。
import { readonly, ref } from 'vue'

export type RouteName = 'login' | 'workbench'

function parseHash(hash: string): RouteName {
  return hash.startsWith('#/workbench') ? 'workbench' : 'login'
}

const current = ref<RouteName>(parseHash(window.location.hash))

window.addEventListener('hashchange', () => {
  current.value = parseHash(window.location.hash)
})

export function useRouter() {
  function navigate(name: RouteName): void {
    const target = name === 'workbench' ? '#/workbench' : '#/login'
    if (window.location.hash !== target) {
      window.location.hash = target
    }
    current.value = name
  }

  return { current: readonly(current), navigate }
}
