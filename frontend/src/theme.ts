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

export const layout = {
  siderWidth: 208,
  siderTheme: 'dark' as 'dark' | 'light',
}
