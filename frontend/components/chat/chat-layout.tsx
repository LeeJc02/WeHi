'use client'

import { SidebarNav } from './sidebar-nav'
import { ConversationList } from './conversation-list'
import { ChatWindow } from './chat-window'
import { ContactsPanel } from './contacts-panel'
import { SettingsPanel } from './settings-panel'
import { useChatStore } from '@/lib/chat-store'

export function ChatLayout() {
  const { activeView } = useChatStore()

  return (
    <div className="chat-shell-background relative isolate h-screen w-full overflow-hidden bg-background">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(255,255,255,0.18),transparent_28%),radial-gradient(circle_at_left_bottom,rgba(118,171,255,0.16),transparent_30%),linear-gradient(180deg,rgba(8,12,22,0.28),rgba(8,12,22,0.54))]" />
      <div className="absolute inset-0 backdrop-blur-2xl" />
      <div className="relative z-10 flex h-full w-full overflow-hidden">
        {/* 左侧导航栏 */}
        <SidebarNav />
        
        {/* 中间面板 */}
        <div className="w-72 min-w-[280px] border-r border-panel-border/55 bg-panel/58 backdrop-blur-xl flex-shrink-0 shadow-[0_0_0_1px_rgba(255,255,255,0.08)]">
          {activeView === 'chat' && <ConversationList />}
          {activeView === 'contacts' && <ContactsPanel />}
          {activeView === 'settings' && <SettingsPanel />}
        </div>
        
        {/* 右侧主内容区 */}
        <div className="flex-1 min-w-0 bg-white/8 backdrop-blur-sm">
          {activeView === 'chat' && <ChatWindow />}
          {activeView === 'contacts' && (
            <div className="h-full flex items-center justify-center text-muted-foreground">
              <div className="rounded-2xl border border-white/20 bg-white/12 px-8 py-6 text-center shadow-lg backdrop-blur-xl">
                <p>选择一个好友或群组开始聊天</p>
              </div>
            </div>
          )}
          {activeView === 'settings' && (
            <div className="h-full flex items-center justify-center text-muted-foreground">
              <p>设置功能开发中...</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
