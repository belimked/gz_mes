// Shared Ant Design dark theme for the "tech" login styles: translucent inputs on dark backgrounds.
import { theme as antTheme, type ConfigProviderProps } from 'ant-design-vue'

export function techTheme(primary: string): ConfigProviderProps['theme'] {
  return {
    algorithm: antTheme.darkAlgorithm,
    token: {
      colorPrimary: primary,
      borderRadius: 8,
      colorBgContainer: 'rgba(255,255,255,0.04)',
      colorBorder: 'rgba(255,255,255,0.14)',
      colorTextPlaceholder: 'rgba(255,255,255,0.35)',
    },
  }
}
