<script setup>
import { ref, onMounted, watch, nextTick, computed } from 'vue'
import axios from '../services/axios.js'
import UserSearch from './UserSearch.vue'

const props = defineProps({
    conversationId: {
        type: [String, Number],
        required: true
    }
})

const conversation = ref(null)
const messages = ref([])
const newMessage = ref('')
const loading = ref(false)
const sending = ref(false)
const messagesContainer = ref(null)

// UI States
const showAddMember = ref(false)
const showForwardModal = ref(false)
const messageToForward = ref(null)
const forwardTargets = ref([]) // List of conversations to forward to
const groupPhotoInput = ref(null)

const replyingTo = ref(null)
const selectedImage = ref(null)
const chatImageInput = ref(null)

const pollTimeout = ref(null)
const isPolling = ref(false)

async function loadData(isInitial = false) {
    if (!props.conversationId) return
    try {
        const response = await axios.get(`/conversations/${props.conversationId}`, { params: { _t: Date.now() } })
        
        const newMessages = response.data.messages
        // Auto-scroll if new messages arrived
        if (newMessages.length > messages.value.length) {
            scrollToBottom()
        }
        
        conversation.value = response.data
        messages.value = newMessages
    } catch (e) {
        console.error("Error loading conversation:", e)
    } finally {
        if (isPolling.value) {
            pollTimeout.value = setTimeout(() => loadData(false), 2000)
        }
    }
}

async function sendMessage() {
    if (!newMessage.value.trim() && !selectedImage.value) return
    sending.value = true
    try {
        let payload
        let isMultipart = false

        if (selectedImage.value) {
            isMultipart = true
            payload = new FormData()
            payload.append('content', newMessage.value)
            payload.append('type', 'image')
            payload.append('image', selectedImage.value)
            if (replyingTo.value) {
                payload.append('replyToId', replyingTo.value.id)
            }
        } else {
            payload = {
                type: 'text',
                content: newMessage.value,
                replyToId: replyingTo.value ? replyingTo.value.id : null
            }
        }

        await axios.post(`/conversations/${props.conversationId}/messages`, payload, {
            headers: isMultipart ? { 'Content-Type': 'multipart/form-data' } : {}
        })

        newMessage.value = ''
        selectedImage.value = null
        replyingTo.value = null
        
        await loadData()
        scrollToBottom() 
    } catch (e) {
        alert("Error sending message: " + e.toString())
    } finally {
        sending.value = false
    }
}

function selectImage(event) {
    const file = event.target.files[0]
    if (file) selectedImage.value = file
}
function clearImage() {
    selectedImage.value = null
}
function setReply(msg) {
    replyingTo.value = msg
}
function clearReply() {
    replyingTo.value = null
}

async function deleteMessage(id) {
    if (!confirm("Delete this message?")) return
    try {
        await axios.delete(`/messages/${id}`)
        await loadData()
    } catch (e) {
        alert("Error deleting: " + e.toString())
    }
}

// Group Management
const emit = defineEmits(['conversation-updated'])

async function addMember(user) {
    try {
        await axios.post(`/groups/${props.conversationId}/members`, { userId: user.id })
        showAddMember.value = false
        alert(`Added ${user.username} to group!`)
        await loadData()
        emit('conversation-updated')
    } catch (e) {
        alert("Error adding member: " + e.toString())
    }
}

async function leaveGroup() {
    if (!confirm("Are you sure you want to leave this group?")) return
    try {
        await axios.post(`/groups/${props.conversationId}/leave`)
        window.location.reload()
    } catch (e) {
        alert("Error leaving group: " + e.toString())
    }
}

async function updateGroupName() {
    const newName = prompt("Enter new group name:", conversation.value.name)
    if (!newName) return
    try {
        await axios.put(`/groups/${props.conversationId}/name`, { name: newName })
        await loadData()
        emit('conversation-updated')
    } catch (e) {
        alert("Error updating name: " + e.toString())
    }
}

function triggerPhotoUpload() {
    groupPhotoInput.value.click()
}

async function updateGroupPhoto(event) {
    const file = event.target.files[0]
    if (!file) return

    if (file.type !== 'image/png' && file.type !== 'image/jpeg') {
        alert("Only PNG or JPEG allowed")
        return
    }

    try {
        await axios.put(`/groups/${props.conversationId}/photo`, file, {
            headers: {
                'Content-Type': file.type
            }
        })
        await loadData()
        emit('conversation-updated')
        alert("Group photo updated!")
    } catch (e) {
        alert("Error updating photo: " + e.toString())
    } finally {
        event.target.value = ''
    }
}

// Reactions
async function reactToMessage(msgId, emoji) {
    try {
        await axios.post(`/messages/${msgId}/reactions`, { emoji })
        await loadData()
    } catch (e) {
        alert("Error reacting: " + e.toString())
    }
}

async function removeReaction(msgId, reactionId) {
    try {
        await axios.delete(`/messages/${msgId}/reactions/${reactionId}`)
        await loadData()
    } catch (e) {
        alert("Error removing reaction: " + e.toString())
    }
}

// Forwarding
async function openForwardModal(msg) {
    messageToForward.value = msg
    showForwardModal.value = true
    try {
        const res = await axios.get('/conversations')
        forwardTargets.value = res.data
    } catch (e) {
        console.error("Error loading forward targets", e)
    }
}

async function forwardMessage(targetConvId) {
    if (!messageToForward.value) return
    try {
        await axios.post(`/messages/${messageToForward.value.id}/forward`, {
            conversationId: targetConvId
        })
        showForwardModal.value = false
        messageToForward.value = null
        alert("Message forwarded!")
    } catch (e) {
        alert("Error forwarding: " + e.toString())
    }
}

function scrollToBottom() {
    nextTick(() => {
        if (messagesContainer.value) {
            messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
        }
    })
}

function startPolling() {
    if (isPolling.value) return
    isPolling.value = true
    loadData(true)
}

function stopPolling() {
    isPolling.value = false
    if (pollTimeout.value) {
        clearTimeout(pollTimeout.value)
        pollTimeout.value = null
    }
}

// Refresh list in parent component to update unread counts/last message
watch(conversation, () => {
   emit('conversation-updated')
})

watch(() => props.conversationId, (newId) => {
    stopPolling()
    if (newId) {
        conversation.value = null
        messages.value = []
        startPolling()
    }
})

onMounted(() => {
    if (props.conversationId) startPolling()
})
import { onUnmounted } from 'vue'
onUnmounted(() => stopPolling())

</script>

<template>
    <div class="d-flex flex-column h-100 position-relative">
        <!-- Header -->
        <div class="p-3 border-bottom bg-white d-flex justify-content-between align-items-center" v-if="conversation">
            <div class="d-flex align-items-center">
                 <!-- Avatar -->
                <div class="me-3 position-relative">
                    <img v-if="conversation.photo" :src="`data:image/png;base64,${conversation.photo}`" class="rounded-circle" width="40" height="40" style="object-fit: cover;">
                    <div v-else class="rounded-circle bg-secondary d-flex align-items-center justify-content-center text-white" style="width: 40px; height: 40px;">
                        {{ (conversation.name || '?').charAt(0).toUpperCase() }}
                    </div>
                </div>
                <div>
                    <h5 class="mb-0">
                        {{ conversation.name }}
                    </h5>
                <small class="text-muted" v-if="conversation.isGroup" :title="conversation.members.map(m => m.username).join(', ')">
                    {{ conversation.members.length }} members
                </small>
            </div>
            </div>
            <div class="d-flex gap-2">
                <template v-if="conversation.isGroup">
                    <!-- Photo Upload Input (Hidden) -->
                    <input type="file" ref="groupPhotoInput" class="d-none" accept="image/png, image/jpeg" @change="updateGroupPhoto">
                    
                    <button class="btn btn-sm btn-outline-secondary" @click="triggerPhotoUpload" title="Change Group Photo">
                        📷
                    </button>
                    <button class="btn btn-sm btn-outline-secondary" @click="updateGroupName" title="Edit Name">
                        ✎
                    </button>
                    <button class="btn btn-sm btn-primary" @click="showAddMember = !showAddMember" title="Add Member">
                        + User
                    </button>
                    <button class="btn btn-sm btn-outline-danger" @click="leaveGroup" title="Leave Group">
                        Leave
                    </button>
                </template>
            </div>
        </div>

        <!-- Add Member Panel -->
        <div v-if="showAddMember" class="p-3 bg-light border-bottom">
            <h6>Add User to Group</h6>
            <UserSearch @select-user="addMember" />
        </div>

        <!-- Messages Area -->
        <div class="flex-grow-1 overflow-auto p-3 bg-white" ref="messagesContainer">
            <div v-if="messages.length === 0" class="text-center text-muted mt-5">
                No messages yet. Say hello!
            </div>
            
            <div 
                v-for="msg in messages" 
                :key="msg.id" 
                class="d-flex mb-3"
                :class="{ 'justify-content-end': msg.senderId == localStorageProp }" 
            >
                <div 
                    class="card shadow-sm position-relative group-hover-container"
                    :class="msg.senderId == localStorageProp ? 'bg-primary text-white' : 'bg-light'"
                    style="max-width: 75%; min-width: 200px;"
                >
                    <div class="card-body py-2 px-3">
                        <!-- Sender Name (if group and not me) -->
                        <div v-if="conversation?.isGroup && msg.senderId != localStorageProp" class="text-muted small fw-bold mb-1">
                             {{ conversation.members.find(m => m.id == msg.senderId)?.username || 'Unknown' }}
                        </div>

                        <!-- Reply Snippet -->
                        <div v-if="msg.replyTo" class="mb-2 p-2 rounded border-start border-4 border-primary bg-white bg-opacity-75 small">
                            <div class="fw-bold">{{ msg.replyTo.senderId == localStorageProp ? 'You' : conversation.members.find(m => m.id == msg.replyTo.senderId)?.username || 'Unknown' }}</div>
                            <div class="text-truncate">{{ msg.replyTo.content }}</div>
                        </div>

                        <!-- Message Content -->
                        <p class="mb-1">{{ msg.content }}</p>

                        <!-- Attached Image -->
                        <div v-if="msg.photo" class="mt-2">
                             <img :src="`data:image/jpeg;base64,${msg.photo}`" class="img-fluid rounded" style="max-height: 200px;">
                        </div>
                        
                        <!-- Reactions -->
                        <div v-if="msg.reactions && msg.reactions.length > 0" class="d-flex gap-1 flex-wrap mt-2">
                             <span 
                                v-for="reaction in msg.reactions" 
                                :key="reaction.id"
                                class="badge bg-secondary rounded-pill cursor-pointer"
                                :class="{ 'bg-success': reaction.userId == localStorageProp }"
                                @click="reaction.userId == localStorageProp ? removeReaction(msg.id, reaction.id) : null"
                                :title="'Reacted by: ' + (reaction.userId == localStorageProp ? 'You' : (conversation.members.find(m => m.id == reaction.userId)?.username || 'Unknown'))"
                             >
                                {{ reaction.emoji }}
                             </span>
                        </div>

                        <div class="d-flex justify-content-between align-items-center mt-1">
                             <div class="d-flex align-items-center gap-1">
                                <small class="opacity-75" style="font-size: 0.7em;">
                                    {{ new Date(msg.timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'}) }}
                                </small>
                                <!-- Read Receipts -->
                                <span v-if="msg.senderId == localStorageProp" style="font-size: 0.8em; line-height: 1;">
                                    <span v-if="msg.read" class="text-white">✓✓</span>
                                    <span v-else-if="msg.received" class="text-white-50">✓</span>
                                    <span v-else class="text-white-50">🕒</span>
                                </span>
                             </div>
                            
                            <!-- Actions -->
                            <div class="d-flex gap-2">
                                <!-- React -->
                                <div class="dropdown">
                                    <button class="btn btn-sm p-0 opacity-50 hover-opacity-100 text-reset" data-bs-toggle="dropdown">
                                        😀
                                    </button>
                                    <ul class="dropdown-menu p-1" style="min-width: unset;">
                                        <li class="d-flex gap-1">
                                            <button class="btn btn-sm btn-link text-decoration-none" @click="reactToMessage(msg.id, '👍')">👍</button>
                                            <button class="btn btn-sm btn-link text-decoration-none" @click="reactToMessage(msg.id, '❤️')">❤️</button>
                                            <button class="btn btn-sm btn-link text-decoration-none" @click="reactToMessage(msg.id, '😂')">😂</button>
                                        </li>
                                    </ul>
                                </div>
                                
                                <!-- Reply -->
                                <button 
                                    class="btn btn-sm p-0 opacity-50 hover-opacity-100 text-reset" 
                                    @click="setReply(msg)"
                                    title="Reply"
                                >
                                    ↩
                                </button>
                                
                                <!-- Forward -->
                                <button 
                                    class="btn btn-sm p-0 opacity-50 hover-opacity-100 text-reset" 
                                    @click="openForwardModal(msg)"
                                    title="Forward"
                                >
                                    ➥
                                </button>

                                <!-- Delete -->
                                <button 
                                    v-if="msg.senderId == localStorageProp" 
                                    class="btn btn-sm btn-link text-danger text-decoration-none p-0" 
                                    @click="deleteMessage(msg.id)"
                                    title="Delete"
                                >
                                    &times;
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Input Area -->
        <div class="p-3 border-top bg-light">
             <!-- Context Banner (Reply / Image) -->
             <div v-if="replyingTo" class="mb-2 p-2 bg-white rounded border-start border-4 border-primary d-flex justify-content-between align-items-center small">
                <div>
                    <strong>Replying to {{ replyingTo.senderId == localStorageProp ? 'You' : conversation.members.find(m => m.id == replyingTo.senderId)?.username }}:</strong>
                    <span class="text-muted ms-1 text-truncate d-inline-block" style="max-width: 200px;">{{ replyingTo.content }}</span>
                </div>
                <button type="button" class="btn-close btn-close-white small" @click="clearReply" aria-label="Close"></button>
             </div>
             <div v-if="selectedImage" class="mb-2 p-2 bg-white rounded d-flex justify-content-between align-items-center">
                <span class="small text-muted">Image selected: {{ selectedImage.name }}</span>
                <button type="button" class="btn-close small" @click="clearImage" aria-label="Close"></button>
             </div>

            <form @submit.prevent="sendMessage" class="d-flex gap-2">
                 <!-- Image Upload -->
                <input type="file" ref="chatImageInput" class="d-none" accept="image/png, image/jpeg" @change="selectImage">
                <button type="button" class="btn btn-outline-secondary" @click="chatImageInput.click()" title="Attach Image">
                    📎
                </button>
                
                <input 
                    type="text" 
                    class="form-control" 
                    v-model="newMessage" 
                    placeholder="Type a message..." 
                    :disabled="sending"
                >
                <button type="submit" class="btn btn-primary" :disabled="sending || (!newMessage.trim() && !selectedImage)">
                    Send
                </button>
            </form>
        </div>

        <!-- Forward Modal (Simple Overlay) -->
        <div v-if="showForwardModal" class="position-absolute top-0 start-0 w-100 h-100 bg-white d-flex flex-column p-3" style="z-index: 1000;">
             <div class="d-flex justify-content-between mb-3">
                <h5>Forward Message</h5>
                <button class="btn btn-close" @click="showForwardModal = false"></button>
            </div>
            <p>Select a conversation to forward to:</p>
            <div class="list-group overflow-auto">
                <button 
                    v-for="target in forwardTargets" 
                    :key="target.id" 
                    class="list-group-item list-group-item-action"
                    @click="forwardMessage(target.id)"
                >
                    {{ target.name }}
                </button>
            </div>
        </div>

    </div>
</template>

<script>
export default {
    computed: {
        localStorageProp() {
            return localStorage.getItem('token')
        }
    }
}
</script>

<style scoped>
.hover-opacity-100:hover {
    opacity: 1 !important;
}
.cursor-pointer {
    cursor: pointer;
}
</style>
