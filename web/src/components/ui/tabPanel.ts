export function tabPanelProps(idPrefix: string, value: string) {
  return {
    id: `${idPrefix}-panel-${value}`,
    role: 'tabpanel' as const,
    'aria-labelledby': `${idPrefix}-tab-${value}`,
    tabIndex: 0,
  }
}
