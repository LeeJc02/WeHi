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
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(255,255,255,0.2),transparent_24%),radial-gradient(circle_at_left_bottom,rgba(52,211,153,0.16),transparent_28%),linear-gradient(180deg,rgba(8,12,22,0.18),rgba(8,12,22,0.46))]" />
      <div className="absolute inset-0 backdrop-blur-[18px]" />
      <div className="relative z-10 flex h-full w-full overflow-hidden p-3">
        {/* 左侧导航栏 */}
        <SidebarNav />
        
        {/* 中间面板 */}
        <div className="ml-3 w-72 min-w-[280px] flex-shrink-0 overflow-hidden rounded-[28px] border border-white/14 bg-[linear-gradient(180deg,rgba(255,255,255,0.88),rgba(245,248,246,0.76))] shadow-[0_16px_42px_rgba(15,23,42,0.16)] backdrop-blur-xl">
          {activeView === 'chat' && <ConversationList />}
          {activeView === 'contacts' && <ContactsPanel />}
          {activeView === 'settings' && <SettingsPanel />}
        </div>
        
        {/* 右侧主内容区 */}
        <div className="ml-3 min-w-0 flex-1 overflow-hidden rounded-[30px] border border-white/12 bg-[linear-gradient(180deg,rgba(255,255,255,0.2),rgba(255,255,255,0.08))] shadow-[0_20px_44px_rgba(15,23,42,0.18)] backdrop-blur-xl">
          {activeView === 'chat' && <ChatWindow />}
          {activeView === 'contacts' && (
            <div className="h-full flex items-center justify-center text-muted-foreground">
              <div className="rounded-[28px] border border-white/20 bg-white/14 px-8 py-6 text-center shadow-lg backdrop-blur-xl">
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
