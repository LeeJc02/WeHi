import type { Metadata, Viewport } from 'next'
import { AuthProvider } from '@/lib/auth-context'
import { ChatStoreProvider } from '@/lib/chat-store'
import { ThemeProvider } from '@/components/providers/theme-provider'
import { Toaster } from '@/components/ui/sonner'
import './globals.css'

export const metadata: Metadata = {
  title: {
    default: 'WeHi',
    template: '%s · WeHi',
  },
  applicationName: 'WeHi',
  description: 'A polished distributed IM client built for the WeHi messaging stack.',
  keywords: ['WeHi', 'instant messaging', 'distributed IM', 'Go', 'WebSocket', 'Next.js'],
  icons: {
    icon: '/wehi-icon.png',
    shortcut: '/wehi-icon.png',
    apple: '/apple-icon.png',
  },
}

export const viewport: Viewport = {
  themeColor: '#07c160',
  width: 'device-width',
  initialScale: 1,
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <body className="font-sans antialiased overflow-hidden">
        <ThemeProvider>
          <AuthProvider>
            <ChatStoreProvider>
              {children}
              <Toaster />
            </ChatStoreProvider>
          </AuthProvider>
        </ThemeProvider>
      </body>
    </html>
  )
}
