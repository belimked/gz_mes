// Central place for look & feel. Change colors, radius, brand text and layout here;
// views only use Ant Design Vue components and these tokens. See docs/design/ui.md.
import type { ConfigProviderProps } from 'ant-design-vue'

export const brand = {
  name: 'GZ MES',
  subtitle: '铝单板生产执行系统',
}

export const theme: ConfigProviderProps['theme'] = {
  token: {
    colorPrimary: '#1677FF',
    colorSuccess: '#52C41A',
    colorWarning: '#FAAD14',
    colorError: '#FF4D4F',
    colorBgLayout: '#F5F7FA',
    colorText: '#1F1F1F',
    colorTextSecondary: '#595959',
    borderRadius: 6,
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif',
  },
}

/** Semantic status colours for tags and process cards (ui.md §2). */
export const statusTone = {
  neutral: { bg: '#FAFAFA', border: '#D9D9D9', text: '#595959' },
  processing: { bg: '#E6F4FF', border: '#91CAFF', text: '#0958D9' },
  success: { bg: '#F6FFED', border: '#B7EB8F', text: '#389E0D' },
  warning: { bg: '#FFFBE6', border: '#FFE58F', text: '#D48806' },
  error: { bg: '#FFF1F0', border: '#FFA39E', text: '#CF1322' },
} as const
export type Tone = keyof typeof statusTone

/** The 11 shop-floor steps shown in pipelines and timelines (ui.md §6). */
export const PROCESS_STEPS = [
  '开料',
  '折弯',
  '钣金下线',
  '钣金质检',
  '喷涂接收',
  '上挂喷涂',
  '下挂冷却',
  '喷涂质检',
  '成品接收',
  '成品装架',
  '发货对账',
] as const

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
  siderTheme: 'light' as 'dark' | 'light',
  /** Default login page style. */
  loginStyle: 'aurora' as LoginStyle,
}
