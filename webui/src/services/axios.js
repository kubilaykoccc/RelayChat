import axios from "axios";

var apiURL = __API_URL__;
// Dynamic override: If running on a real IP (not localhost), assume API is on the same host.
// This allows the user to test from Mac (using 192.168.x.x) while checking in 'localhost' for the specific requirements.
if (window.location.hostname !== 'localhost' && window.location.hostname !== '127.0.0.1') {
	apiURL = `http://${window.location.hostname}:3000`;
}

const instance = axios.create({
	baseURL: apiURL,
	timeout: 1000 * 5
});

instance.interceptors.request.use(
	(config) => {
		const token = localStorage.getItem('token');
		if (token) {
			config.headers['Authorization'] = `Bearer ${token}`;
		}
		return config;
	},
	(error) => {
		return Promise.reject(error);
	}
);

export default instance;
