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
  { key: 'aurora', label: '极光玻璃' },
  { key: 'nebula', label: '星云紫' },
  { key: 'tide', label: '极光帘幕' },
  { key: 'ember', label: '暖色余晖' },
] as const
export type LoginStyle = (typeof loginStyles)[number]['key']
export const isLoginStyle = (s: unknown): s is LoginStyle => loginStyles.some((x) => x.key === s)

export const layout = {
  siderWidth: 208,
  siderTheme: 'dark' as 'dark' | 'light',
  /** Default login page style. */
  loginStyle: 'aurora' as LoginStyle,
}
