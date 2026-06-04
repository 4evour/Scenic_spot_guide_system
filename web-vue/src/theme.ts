import { type GlobalThemeOverrides } from 'naive-ui'

export const adminThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#63e2b7',
    primaryColorHover: '#7ee8c7',
    primaryColorPressed: '#4fd1a0',
    primaryColorSuppl: '#63e2b7',
    borderRadius: '8px',
    borderRadiusSmall: '6px',
    fontFamily: '-apple-system, "PingFang SC", "Segoe UI", sans-serif',
  },
  Button: {
    borderRadiusMedium: '8px',
  },
  Card: {
    borderRadius: '12px',
  },
  DataTable: {
    borderRadius: '8px',
  },
  Input: {
    borderRadius: '8px',
  },
}
