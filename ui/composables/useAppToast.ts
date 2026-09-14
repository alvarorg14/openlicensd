export const useAppToast = () => {
  const toast = useToast()

  const success = (title: string, description?: string) => {
    toast.add({
      title,
      description,
      color: 'success',
      icon: 'i-lucide-check'
    })
  }

  return {
    success
  }
}
