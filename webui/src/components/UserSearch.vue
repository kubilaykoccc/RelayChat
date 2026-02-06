<script setup>
import { ref } from "vue";
import axios from "../services/axios.js";

const searchQuery = ref("");
const users = ref([]);
const loading = ref(false);
const error = ref(null);

const emit = defineEmits(["select-user"]);

async function search() {
	if (searchQuery.value.length < 3) return;
	console.log("Searching for:", searchQuery.value); // DEBUG
	loading.value = true;
	error.value = null;
	try {
		const response = await axios.get("/users/search", {
			params: { q: searchQuery.value },
		});
		console.log("Search response:", response.data); // DEBUG
		users.value = response.data;
	} catch (e) {
		console.error("Search error:", e); // DEBUG
		error.value = e.toString();
	} finally {
		loading.value = false;
	}
}

function selectUser(user) {
	emit("select-user", user);
}
</script>

<template>
	<div class="p-3">
		<label class="form-label small text-muted">New Chat</label>
		<div class="input-group mb-2">
			<input
				type="text"
				class="form-control form-control-sm"
				v-model="searchQuery"
				@keyup.enter="search"
				placeholder="Search user..."
			/>
			<button
				class="btn btn-outline-secondary btn-sm"
				@click="search"
				:disabled="loading"
			>
				<span
					v-if="loading"
					class="spinner-border spinner-border-sm"
				></span>
				<span v-else>Go</span>
			</button>
		</div>

		<div v-if="users.length > 0" class="list-group">
			<button
				v-for="user in users"
				:key="user.id"
				class="list-group-item list-group-item-action py-2 px-3 small"
				@click="selectUser(user)"
			>
				@{{ user.username }}
			</button>
		</div>
		<div
			v-else-if="!loading && searchQuery.length >= 3"
			class="text-muted small text-center"
		>
			No users found.
		</div>
	</div>
</template>
