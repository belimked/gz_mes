// Central place for look & feel. Change colors, radius, brand text and layout here;
// views only use Ant Design Vue components and these tokens.
import type { ConfigProviderProps } from 'ant-design-vue'

export const brand = {
  name: 'GZ MES',
  subtitle: '铝单板生产执行系统',
}

export const theme: ConfigProviderProps['theme'] = {
  token: {
    colorPrimary: '#1677ff',
    borderRadius: 4,
  },
}

/** Available login page styles. Preview any of them with /login?style=<key>. */
export const loginStyles = [
  { key: 'split', label: '蓝色分栏' },
  { key: 'card', label: '居中卡片' },
  { key: 'dark', label: '深色科技' },
  { key: 'industrial', label: '工业橙' },
  { key: 'minimal', label: '极简白' },
  { key: 'aurora', label: '极光玻璃' },
  { key: 'twin', label: '数字孪生' },
  { key: 'flow', label: '粒子数据流' },
  { key: 'bento', label: 'Bento 网格' },
  { key: 'holo', label: '全息线框' },
] as const
export type LoginStyle = (typeof loginStyles)[number]['key']
export const isLoginStyle = (s: unknown): s is LoginStyle => loginStyles.some((x) => x.key === s)

export const layout = {
  siderWidth: 208,
  siderTheme: 'dark' as 'dark' | 'light',
  /** Default login page style. */
  loginStyle: 'split' as LoginStyle,
}
