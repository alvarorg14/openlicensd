export const resourceTypeLabels: Record<string, string> = {
  license: 'License',
  product: 'Product',
  policy: 'Policy',
  machine: 'Machine',
  user: 'User',
  api_token: 'API token',
  session: 'Session'
}

export const resourceTypeBadgeColors: Record<string, string> = {
  license: 'primary',
  product: 'info',
  policy: 'success',
  machine: 'warning',
  user: 'secondary',
  api_token: 'neutral',
  session: 'neutral'
}

export const actionLabels: Record<string, string> = {
  'license.create': 'Create',
  'license.update': 'Update',
  'license.delete': 'Delete',
  'license.revoke': 'Revoke',
  'license.unrevoke': 'Unrevoke',
  'product.create': 'Create',
  'product.update': 'Update',
  'product.delete': 'Delete',
  'policy.create': 'Create',
  'policy.update': 'Update',
  'policy.delete': 'Delete',
  'machine.update': 'Rename',
  'machine.release': 'Release',
  'user.create': 'Create',
  'user.update': 'Update',
  'user.delete': 'Delete',
  'user.password_set': 'Set password',
  'user.disable': 'Disable',
  'user.enable': 'Enable',
  'auth.password_change': 'Change password',
  'auth.logout': 'Logout',
  'api_token.create': 'Create',
  'api_token.revoke': 'Revoke',
  'api_token.delete': 'Delete'
}

export const actionsByResourceType: Record<string, string[]> = {
  license: ['license.create', 'license.update', 'license.delete', 'license.revoke', 'license.unrevoke'],
  product: ['product.create', 'product.update', 'product.delete'],
  policy: ['policy.create', 'policy.update', 'policy.delete'],
  machine: ['machine.update', 'machine.release'],
  user: [
    'user.create',
    'user.update',
    'user.delete',
    'user.password_set',
    'user.disable',
    'user.enable',
    'auth.password_change'
  ],
  api_token: ['api_token.create', 'api_token.revoke', 'api_token.delete']
}

export function formatActionLabel(action: string): string {
  return actionLabels[action] ?? action
}

export function formatResourceTypeLabel(resourceType: string): string {
  return resourceTypeLabels[resourceType] ?? resourceType
}
