<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from '../services/axios.js'

const router = useRouter()
const username = ref('')
const loading = ref(false)
const error = ref(null)

async function doLogin() {
    loading.value = true
    error.value = null
    
    if (username.value.length < 3) {
        error.value = "Username must be at least 3 characters."
        loading.value = false
        return
    }

    try {
        const response = await axios.post('/session', { name: username.value })
        const identifier = response.data.identifier
        
        // Save to localStorage
        localStorage.setItem('token', identifier)
        localStorage.setItem('username', username.value)
        
        // Redirect
        router.push('/')
    } catch (e) {
        if (e.response && e.response.data) {
            error.value = e.response.data
        } else {
            error.value = e.toString()
        }
    } finally {
        loading.value = false
    }
}
</script>

<template>
    <div class="d-flex justify-content-center align-items-center vh-100 bg-light">
        <div class="card p-4 shadow-sm" style="max-width: 400px; width: 100%;">
            <h2 class="text-center mb-4">WASAText Login</h2>
            <form @submit.prevent="doLogin">
                <div class="mb-3">
                    <label for="username" class="form-label">Username</label>
                    <input type="text" class="form-control" id="username" v-model="username" placeholder="Enter your name" required minlength="3" maxlength="16">
                </div>
                <div v-if="error" class="alert alert-danger">{{ error }}</div>
                <button type="submit" class="btn btn-primary w-100" :disabled="loading">
                    <span v-if="loading" class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                    Login
                </button>
            </form>
        </div>
    </div>
</template>

<style scoped>
</style>
