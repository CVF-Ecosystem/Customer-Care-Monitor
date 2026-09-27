// CCMAI-UX-000: shared component types.

export interface ActionMenuItem {
  key: string
  label: string
  icon?: string
  // Destructive: listed last, shown in the danger color; the parent must confirm before acting.
  danger?: boolean
}
