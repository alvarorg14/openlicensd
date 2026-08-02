export default defineAppConfig({
  ui: {
    colors: {
      primary: 'brand',
      neutral: 'navy'
    },
    input: { slots: { root: 'w-full' } },
    inputNumber: { slots: { root: 'w-full' } },
    inputMenu: { slots: { root: 'w-full' } },
    textarea: { slots: { root: 'w-full' } },
    select: { slots: { base: 'w-full' } },
    selectMenu: { slots: { base: 'w-full' } }
  }
})
