import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '../api'

export interface Conversation {
  id: string
  channel_id: string
  channel_name: string
  channel_type: string
  customer_name: string
  last_message_at: string | null
  message_count: number
  created_at: string
}

export interface Message {
  id: string
  sender_type: string
  sender_name: string
  content: string
  content_type: string
  attachments: string
  sent_at: string
}

export const useConversationStore = defineStore('conversations', () => {
  const conversations = ref<Conversation[]>([])
  const messages = ref<Message[]>([])
  const total = ref(0)
  const currentConversation = ref<{ id: string; customer_name: string; message_count: number } | null>(null)

  // CCMAI-RUNTIME-088 (F06): every publication into the refs above is bound to the generation of the
  // request that produced it. A newer request (or a reset) supersedes every older one, so a late reply
  // - success or failure - can neither overwrite current data nor be reported to its caller as current.
  // `owner` identifies the screen instance that issued the request: cleanup by an old owner must not
  // invalidate or clear work that a newer owner started afterwards.
  let listGeneration = 0
  let detailGeneration = 0
  let listOwner: object | undefined
  let detailOwner: object | undefined

  /** Resolves true when this call published its result, false when a newer call or a reset superseded it. */
  async function fetchConversations(tenantId: string, params: Record<string, string | number> = {}, owner?: object): Promise<boolean> {
    const generation = ++listGeneration
    listOwner = owner
    try {
      const { data } = await api.get(`/tenants/${tenantId}/conversations`, { params })
      if (generation !== listGeneration) return false
      conversations.value = data.data
      total.value = data.total
      return true
    } catch (error) {
      if (generation !== listGeneration) return false // a superseded failure is not the caller's current error
      throw error
    }
  }

  /** Same contract as fetchConversations. The previous conversation's data is hidden as soon as the call starts. */
  async function fetchMessages(tenantId: string, conversationId: string, owner?: object): Promise<boolean> {
    const generation = ++detailGeneration
    detailOwner = owner
    messages.value = []
    currentConversation.value = null
    try {
      const { data } = await api.get(`/tenants/${tenantId}/conversations/${conversationId}/messages`)
      if (generation !== detailGeneration) return false
      messages.value = data.messages
      currentConversation.value = data.conversation
      return true
    } catch (error) {
      if (generation !== detailGeneration) return false
      throw error
    }
  }

  /** Drop the selected conversation's data and any pending detail reply. A stale owner is ignored. */
  function resetDetail(owner?: object) {
    if (owner !== undefined && owner !== detailOwner) return
    detailGeneration++
    detailOwner = undefined
    messages.value = []
    currentConversation.value = null
  }

  /** Drop list and detail data plus every pending reply of this owner (tenant change, unmount). */
  function reset(owner?: object) {
    if (owner === undefined || owner === listOwner) {
      listGeneration++
      listOwner = undefined
      conversations.value = []
      total.value = 0
    }
    resetDetail(owner)
  }

  return { conversations, messages, total, currentConversation, fetchConversations, fetchMessages, resetDetail, reset }
})
