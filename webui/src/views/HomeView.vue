<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import ConversationList from '../components/ConversationList.vue'
import ChatWindow from '../components/ChatWindow.vue'
import UserSearch from '../components/UserSearch.vue'
import axios from '../services/axios.js'

const router = useRouter()
const username = localStorage.getItem('username')
const selectedConversationId = ref(null)
const conversationListRef = ref(null)

function doLogout() {
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    router.push('/login')
}

function onSelectConversation(id) {
    selectedConversationId.value = id
}

async function onSelectUser(user) {
    // Create conversation (1-on-1)
    try {
        const response = await axios.post('/conversations', {
            members: [user.id]
        })
        selectedConversationId.value = response.data.id
        refreshList()
    } catch (e) {
        alert("Error creating conversation: " + e.toString())
    }
}

async function createGroup() {
    const name = prompt("Enter group name:")
    if (!name) return
    
    try {
        const response = await axios.post('/conversations', {
            name: name
        })
        selectedConversationId.value = response.data.id
        refreshList()
    } catch (e) {
        alert("Error creating group: " + e.toString())
    }
}

function refreshList() {
    if (conversationListRef.value) {
        conversationListRef.value.loadConversations() // Expose this method in ConversationList?
        // Need to defineExpose in ConversationList
    }
}
</script>

<template>
    <div class="row h-100 g-0">
        <!-- Sidebar -->
        <nav id="sidebarMenu" class="col-md-4 col-lg-3 d-md-block bg-light sidebar collapse border-end">
            <div class="d-flex flex-column h-100">
                <div class="p-3 border-bottom bg-white">
                    <div class="d-flex justify-content-between align-items-center mb-2">
                        <span class="fs-5 fw-bold text-primary">WASAText</span>
                    </div>
                    <button class="btn btn-primary w-100 mb-3 d-flex align-items-center justify-content-center gap-2" @click="createGroup" title="New Group">
                        <span class="fw-bold fs-5">+</span> New Group
                    </button>
                    
                    <div class="d-flex align-items-center">
                        <div class="dropdown w-100">
                             <a href="#" class="d-flex align-items-center text-decoration-none dropdown-toggle text-dark" id="dropdownUser1" data-bs-toggle="dropdown" aria-expanded="false">
                                <strong>{{ username }}</strong>
                            </a>
                            <ul class="dropdown-menu text-small shadow w-100" aria-labelledby="dropdownUser1">
                                <li><RouterLink class="dropdown-item" to="/profile">Profile</RouterLink></li>
                                <li><hr class="dropdown-divider"></li>
                                <li><a class="dropdown-item" href="#" @click.prevent="doLogout">Sign out</a></li>
                            </ul>
                        </div>
                    </div>
                </div>
                
                <!-- Search for new 1-on-1 -->
                <div class="border-bottom bg-white">
                    <UserSearch @select-user="onSelectUser" />
                </div>
                
                <div class="flex-grow-1 overflow-auto">
                    <ConversationList ref="conversationListRef" @select-conversation="onSelectConversation" />
                </div>
            </div>
        </nav>

        <!-- Main Content -->
        <main class="col-md-8 ms-sm-auto col-lg-9 h-100 bg-white">
            <div class="h-100" v-if="selectedConversationId">
                <ChatWindow 
                    :conversationId="selectedConversationId" 
                    @conversation-updated="refreshList"
                />
            </div>
            <div class="h-100 d-flex justify-content-center align-items-center text-muted bg-light" v-else>
                <div class="text-center">
                    <h3>Welcome back, {{ username }}!</h3>
                    <p>Select a chat from the sidebar to start messaging.</p>
                </div>
            </div>
        </main>
    </div>
</template>

<style scoped>
.sidebar {
    position: fixed;
    top: 48px; /* Height of header if exists, or just absolute */
    top: 0;
    bottom: 0;
    left: 0;
    z-index: 100;
    padding: 0;
} 
/* Adjust responsive layout if necessary */
</style>
