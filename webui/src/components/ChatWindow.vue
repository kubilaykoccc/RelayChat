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

// Poll interval
let pollController = null

async function loadData() {
    if (!props.conversationId) return
    try {
        const response = await axios.get(`/conversations/${props.conversationId}`)
        conversation.value = response.data
        messages.value = response.data.messages
        scrollToBottom()
    } catch (e) {
        console.error("Error loading conversation:", e)
    }
}

async function sendMessage() {
    if (!newMessage.value.trim()) return
    sending.value = true
    try {
        await axios.post(`/conversations/${props.conversationId}/messages`, {
            type: 'text',
            content: newMessage.value
        })
        newMessage.value = ''
        await loadData()
    } catch (e) {
        alert("Error sending message: " + e.toString())
    } finally {
        sending.value = false
    }
}

async function deleteMessage(id) {
    if (!confirm("Delete this message?")) return
    try {
        await axios.delete(`/conversations/${props.conversationId}/messages/${id}`)
        await loadData()
    } catch (e) {
        alert("Error deleting: " + e.toString())
    }
}

// Group Management
// Defines emits
const emit = defineEmits(['conversation-updated'])

async function addMember(user) {
    try {
        await axios.put(`/conversations/${props.conversationId}/members/${user.id}`)
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
        await axios.delete(`/conversations/${props.conversationId}/members/me`)
        window.location.reload() // Or emit and close? Reload is safer for now but harsh.
    } catch (e) {
        alert("Error leaving group: " + e.toString())
    }
}

async function updateGroupName() {
    const newName = prompt("Enter new group name:", conversation.value.name)
    if (!newName) return
    try {
        await axios.put(`/conversations/${props.conversationId}/name`, { name: newName })
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
        await axios.put(`/conversations/${props.conversationId}/photo`, file, {
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
        // Reset input
        event.target.value = ''
    }
}

// Reactions
async function reactToMessage(msgId, emoji) {
    try {
        await axios.post(`/conversations/${props.conversationId}/messages/${msgId}/reactions`, { emoji })
        await loadData()
    } catch (e) {
        alert("Error reacting: " + e.toString())
    }
}

async function removeReaction(msgId, reactionId) {
    try {
        await axios.delete(`/conversations/${props.conversationId}/messages/${msgId}/reactions/${reactionId}`)
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
        await axios.post(`/conversations/${targetConvId}/messages/forwarded`, {
            forwardedMessageId: messageToForward.value.id
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
    loadData()
    pollController = setInterval(loadData, 3000)
}

function stopPolling() {
    if (pollController) clearInterval(pollController)
}

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
                <small class="text-muted" v-if="conversation.isGroup">
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

                        <p class="mb-1">{{ msg.content }}</p>
                        
                        <!-- Reactions -->
                        <div v-if="msg.reactions && msg.reactions.length > 0" class="d-flex gap-1 flex-wrap mt-2">
                             <span 
                                v-for="reaction in msg.reactions" 
                                :key="reaction.id"
                                class="badge bg-secondary rounded-pill cursor-pointer"
                                :class="{ 'bg-success': reaction.userId == localStorageProp }"
                                @click="reaction.userId == localStorageProp ? removeReaction(msg.id, reaction.id) : null"
                                title="Click to remove your reaction"
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
                                    class="btn btn-sm p-0 opacity-50 hover-opacity-100 text-reset" 
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
            <form @submit.prevent="sendMessage" class="d-flex gap-2">
                <input 
                    type="text" 
                    class="form-control" 
                    v-model="newMessage" 
                    placeholder="Type a message..." 
                    :disabled="sending"
                >
                <button type="submit" class="btn btn-primary" :disabled="sending || !newMessage.trim()">
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
