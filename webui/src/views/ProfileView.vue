<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from '../services/axios.js'

const router = useRouter()
const username = ref(localStorage.getItem('username') || '')
const newUsername = ref(username.value)
const loading = ref(false)
const msg = ref(null)
const msgType = ref('success') // success or danger

const photoFile = ref(null)

function onFileChange(e) {
    const file = e.target.files[0]
    if (file) {
        photoFile.value = file
    }
}

async function updateProfile() {
    loading.value = true
    msg.value = null
    
    try {
        // Update Username if changed
        if (newUsername.value !== username.value) {
            if (newUsername.value.length < 3) {
                throw new Error("Username too short (min 3 chars)")
            }
            await axios.put('/users/me/name', { name: newUsername.value })
            localStorage.setItem('username', newUsername.value)
            username.value = newUsername.value
        }

        // Update Photo if selected
        if (photoFile.value) {
            // Need to determine content type.
            // API yaml says image/png or image/jpeg as raw binary in body
            const type = photoFile.value.type
            if (type !== 'image/png' && type !== 'image/jpeg') {
                 throw new Error("Only PNG or JPEG allowed")
            }
            
            await axios.put('/users/me/photo', photoFile.value, {
                headers: {
                    'Content-Type': type
                }
            })
        }

        msgType.value = 'success'
        msg.value = "Profile updated successfully!"
        photoFile.value = null // Reset file input
    } catch (e) {
        msgType.value = 'danger'
        msg.value = e.response ? e.response.data : e.toString()
    } finally {
        loading.value = false
    }
}
</script>

<template>
    <div class="container py-5">
        <div class="row justify-content-center">
            <div class="col-md-6">
                <div class="card shadow-sm">
                    <div class="card-header bg-white d-flex justify-content-between align-items-center">
                        <h4 class="mb-0">Profile</h4>
                        <RouterLink to="/" class="btn btn-outline-secondary btn-sm">Back to Chat</RouterLink>
                    </div>
                    <div class="card-body">
                        <div class="mb-3">
                            <label class="form-label">Username</label>
                            <div class="input-group">
                                <span class="input-group-text">@</span>
                                <input type="text" class="form-control" v-model="newUsername" minlength="3" maxlength="16">
                            </div>
                        </div>

                        <div class="mb-3">
                            <label class="form-label">Profile Photo</label>
                            <input class="form-control" type="file" @change="onFileChange" accept="image/png, image/jpeg">
                            <div class="form-text">PNG or JPEG only.</div>
                        </div>
                        
                        <div v-if="msg" :class="['alert', 'alert-' + msgType]">
                            {{ msg }}
                        </div>
                        
                        <button class="btn btn-primary" @click="updateProfile" :disabled="loading">
                            <span v-if="loading" class="spinner-border spinner-border-sm me-2"></span>
                            Update Profile
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
