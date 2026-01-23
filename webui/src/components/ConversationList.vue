<script setup>
import { ref, onMounted } from 'vue'
import axios from '../services/axios.js'

const conversations = ref([])
const loading = ref(false)
const error = ref(null)

const emit = defineEmits(['select-conversation'])

async function loadConversations() {
    loading.value = true
    error.value = null
    try {
        const response = await axios.get('/conversations')
        // Sort by last message timestamp (descending)
        conversations.value = response.data.sort((a, b) => {
            const tA = a.lastMessage ? new Date(a.lastMessage.timestamp) : new Date(0)
            const tB = b.lastMessage ? new Date(b.lastMessage.timestamp) : new Date(0)
            return tB - tA
        })
    } catch (e) {
        error.value = e.toString()
    } finally {
        loading.value = false
    }
}

function selectConversation(id) {
    emit('select-conversation', id)
}

function formatTime(timestamp) {
    if (!timestamp) return ''
    const date = new Date(timestamp)
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function getPhotoSrc(conv) {
    if (conv.photo) {
         // Assuming backend sends raw bytes, we might need to assume it's base64 or blob.
         // Actually, axios/json usually encodes []byte as base64 string.
         // Let's assume standard base64 from Go json marshalling.
         // We need to know the mime type. Assuming png/jpeg.
         return `data:image/png;base64,${conv.photo}` 
         // If it's already a full data URL or URL, this might follow later.
    }
    return null
}

onMounted(() => {
    loadConversations()
})

defineExpose({ loadConversations })
</script>

<template>
    <div v-if="loading" class="text-center p-3">
        <span class="spinner-border spinner-border-sm text-secondary" role="status"></span>
    </div>
    
    <div v-else-if="error" class="alert alert-danger m-3 small">
        {{ error }}
        <button class="btn btn-link btn-sm p-0" @click="loadConversations">Retry</button>
    </div>

    <div v-else class="list-group list-group-flush">
        <button 
            v-for="conv in conversations" 
            :key="conv.id"
            class="list-group-item list-group-item-action py-3 lh-tight"
            @click="selectConversation(conv.id)"
        >
            <div class="d-flex w-100 align-items-center">
                <!-- Avatar -->
                <div class="me-3 position-relative">
                    <img v-if="getPhotoSrc(conv)" :src="getPhotoSrc(conv)" class="rounded-circle" width="48" height="48" style="object-fit: cover;">
                    <div v-else class="rounded-circle bg-secondary d-flex align-items-center justify-content-center text-white" style="width: 48px; height: 48px;">
                        {{ (conv.name || '?').charAt(0).toUpperCase() }}
                    </div>
                </div>

                <div class="flex-grow-1 overflow-hidden">
                     <div class="d-flex w-100 align-items-center justify-content-between">
                        <strong class="mb-1 text-truncate">
                            {{ conv.name }}
                        </strong>
                        <small class="text-muted">{{ conv.lastMessage ? formatTime(conv.lastMessage.timestamp) : '' }}</small>
                    </div>
                    <div class="d-flex w-100 justify-content-between align-items-center">
                        <div class="small text-muted text-truncate" style="max-width: 85%;">
                            <span v-if="conv.lastMessage && conv.lastMessage.type === 'image'">📷 Photo</span>
                            <span v-else>{{ conv.lastMessage ? conv.lastMessage.content : 'No messages yet' }}</span>
                        </div>
                        <span v-if="conv.unreadCount > 0" class="badge bg-primary rounded-pill">{{ conv.unreadCount }}</span>
                    </div>
                </div>
            </div>
        </button>
        
        <div v-if="conversations.length === 0" class="text-center p-4 text-muted small">
            No conversations found.
        </div>
    </div>
</template>

<style scoped>
.list-group-item {
    border-radius: 0;
    cursor: pointer;
}
</style>
