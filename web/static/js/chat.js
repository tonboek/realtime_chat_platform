let ws = null;
let currentUser = null;

// DOM elements
const authForms = document.getElementById('auth-forms');
const chatInterface = document.getElementById('chat-interface');
const loginForm = document.getElementById('loginForm');
const registerForm = document.getElementById('registerForm');
const messagesContainer = document.getElementById('messages');
const messageInput = document.getElementById('messageInput');
const sendBtn = document.getElementById('sendBtn');
const logoutBtn = document.getElementById('logoutBtn');
const profileBtn = document.getElementById('profileBtn');
const onlineUsersContainer = document.getElementById('onlineUsers');
const onlineCountSpan = document.getElementById('onlineCount');
const typingIndicator = document.getElementById('typing-indicator');

// Event listeners
loginForm.addEventListener('submit', handleLogin);
registerForm.addEventListener('submit', handleRegister);
sendBtn.addEventListener('click', sendMessage);
messageInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
        sendMessage();
    }
});
messageInput.addEventListener('input', handleTyping);
logoutBtn.addEventListener('click', handleLogout);
profileBtn.addEventListener('click', goToProfile);

// Check if user is already logged in
document.addEventListener('DOMContentLoaded', function() {
    const savedToken = localStorage.getItem('authToken');
    const savedUsername = localStorage.getItem('username');
    
    if (savedToken && savedUsername) {
        currentUser = savedUsername;
        showChatInterface();
        connectWebSocket();
    }
});

async function handleLogin(e) {
    e.preventDefault();
    const username = document.getElementById('loginUsername').value;
    const password = document.getElementById('loginPassword').value;

    try {
        const response = await fetch('/api/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ username, password }),
        });

        const data = await response.json();
        
        if (response.ok) {
            currentUser = username;
            // Save token and username to localStorage
            localStorage.setItem('authToken', data.token);
            localStorage.setItem('username', data.username);
            showChatInterface();
            connectWebSocket();
        } else {
            alert('Login failed: ' + data.error);
        }
    } catch (error) {
        console.error('Login error:', error);
        alert('Login failed. Please try again.');
    }
}

async function handleRegister(e) {
    e.preventDefault();
    const username = document.getElementById('registerUsername').value;
    const password = document.getElementById('registerPassword').value;

    try {
        const response = await fetch('/api/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ username, password }),
        });

        const data = await response.json();
        
        if (response.ok) {
            alert('Registration successful! Please login.');
            // Switch to login tab
            document.getElementById('login-tab').click();
        } else {
            alert('Registration failed: ' + data.error);
        }
    } catch (error) {
        console.error('Registration error:', error);
        alert('Registration failed. Please try again.');
    }
}

function showChatInterface() {
    authForms.style.display = 'none';
    chatInterface.style.display = 'block';
}

function showAuthForms() {
    authForms.style.display = 'block';
    chatInterface.style.display = 'none';
    currentUser = null;
}

async function loadMessageHistory() {
    try {
        const response = await fetch('/api/messages?limit=50');
        const data = await response.json();
        
        if (response.ok && data.messages) {
            // Clear existing messages
            messagesContainer.innerHTML = '';
            
            // Add historical messages
            data.messages.forEach(msg => {
                addMessage(msg.username, msg.content, msg.created_at, msg.avatar);
            });
            
            console.log(`Loaded ${data.count} messages from history`);
        }
    } catch (error) {
        console.error('Error loading message history:', error);
    }
}

function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const token = localStorage.getItem('authToken');
    const wsUrl = `${protocol}//${window.location.host}/api/ws?token=${encodeURIComponent(token)}`;
    
    ws = new WebSocket(wsUrl);

    ws.onopen = function() {
        console.log('WebSocket connected');
        // Load message history when WebSocket connects
        loadMessageHistory();
        // Load online users
        loadOnlineUsers();
        addMessage('System', 'Connected to chat server', new Date());
    };

    ws.onmessage = function(event) {
        try {
            const data = JSON.parse(event.data);
            
            // Check if it's a typing event
            if (data.type === 'typing_start' || data.type === 'typing_stop') {
                handleTypingEvent(data);
            } else {
                // Regular message
                addMessage(data.username, data.content, data.timestamp, data.avatar);
            }
        } catch (error) {
            console.error('Error parsing message:', error);
        }
    };

    ws.onclose = function() {
        console.log('WebSocket disconnected');
        addMessage('System', 'Disconnected from chat server', new Date());
    };

    ws.onerror = function(error) {
        console.error('WebSocket error:', error);
        addMessage('System', 'Connection error', new Date());
    };
}

function sendMessage() {
    const message = messageInput.value.trim();
    if (!message || !ws || ws.readyState !== WebSocket.OPEN) {
        return;
    }

    const messageData = {
        username: currentUser,
        content: message,
        timestamp: new Date().toISOString()
    };

    ws.send(JSON.stringify(messageData));
    messageInput.value = '';
}

function addMessage(username, message, timestamp, avatar = null) {
    const messageElement = document.createElement('div');
    messageElement.className = 'message';
    
    // Format timestamp
    let timeDisplay = '';
    if (typeof timestamp === 'string') {
        // If timestamp is a string (from WebSocket), parse it
        const date = new Date(timestamp);
        timeDisplay = date.toLocaleString('ru-RU', {
            day: '2-digit',
            month: '2-digit',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    } else {
        // If timestamp is a Date object (from history)
        timeDisplay = timestamp.toLocaleString('ru-RU', {
            day: '2-digit',
            month: '2-digit',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    }
    
    // Default avatar if none provided
    const avatarUrl = avatar || '/static/images/default-avatar.svg';
    
    messageElement.innerHTML = `
        <div class="message-header">
            <img src="${avatarUrl}" alt="Avatar" class="user-avatar" onerror="this.src='/static/images/default-avatar.svg'">
            <div class="message-info">
                <div class="username">${username}</div>
                <div class="timestamp">${timeDisplay}</div>
            </div>
        </div>
        <div class="content">${message}</div>
    `;

    messagesContainer.appendChild(messageElement);
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

async function loadOnlineUsers() {
    try {
        const response = await fetch('/api/users/online');
        const data = await response.json();
        
        if (response.ok && data.users) {
            updateOnlineUsersList(data.users);
        }
    } catch (error) {
        console.error('Error loading online users:', error);
    }
}

function updateOnlineUsersList(users) {
    onlineUsersContainer.innerHTML = '';
    onlineCountSpan.textContent = users.length;
    
    if (users.length === 0) {
        onlineUsersContainer.innerHTML = '<div class="text-muted">Нет пользователей онлайн</div>';
        return;
    }
    
    users.forEach(user => {
        const userElement = document.createElement('div');
        userElement.className = 'online-user';
        
        // Default avatar if none provided
        const avatarUrl = user.avatar || '/static/images/default-avatar.svg';
        
        userElement.innerHTML = `
            <div class="online-user-info">
                <img src="${avatarUrl}" alt="Avatar" class="online-user-avatar" onerror="this.src='/static/images/default-avatar.svg'">
                <span class="status-indicator"></span>
                <span class="online-username">${user.username}</span>
            </div>
        `;
        onlineUsersContainer.appendChild(userElement);
    });
}

let typingTimeout = null;
let isTyping = false;

function handleTyping() {
    if (!ws || ws.readyState !== WebSocket.OPEN || !currentUser) {
        return;
    }

    if (!isTyping) {
        isTyping = true;
        sendTypingEvent(true);
    }

    // Clear existing timeout
    if (typingTimeout) {
        clearTimeout(typingTimeout);
    }

    // Set new timeout to stop typing indicator
    typingTimeout = setTimeout(() => {
        isTyping = false;
        sendTypingEvent(false);
    }, 1000); // Stop typing indicator after 1 second of no input
}

function sendTypingEvent(isTyping) {
    if (!ws || ws.readyState !== WebSocket.OPEN || !currentUser) {
        return;
    }

    const typingEvent = {
        username: currentUser,
        is_typing: isTyping,
        type: isTyping ? 'typing_start' : 'typing_stop'
    };

    ws.send(JSON.stringify(typingEvent));
}

function handleTypingEvent(data) {
    if (data.username === currentUser) {
        return; // Don't show our own typing indicator
    }

    if (data.type === 'typing_start') {
        showTypingIndicator(data.username);
    } else if (data.type === 'typing_stop') {
        hideTypingIndicator(data.username);
    }
}

function showTypingIndicator(username) {
    const typingText = typingIndicator.querySelector('.typing-text');
    typingText.textContent = `${username} is typing...`;
    typingIndicator.style.display = 'block';
}

function hideTypingIndicator(username) {
    const typingText = typingIndicator.querySelector('.typing-text');
    if (typingText.textContent.includes(username)) {
        typingIndicator.style.display = 'none';
    }
}

function handleLogout() {
    if (ws) {
        ws.close();
    }
    // Clear localStorage
    localStorage.removeItem('authToken');
    localStorage.removeItem('username');
    showAuthForms();
    messagesContainer.innerHTML = '';
    messageInput.value = '';
    onlineUsersContainer.innerHTML = '';
    onlineCountSpan.textContent = '0';
    typingIndicator.style.display = 'none';
    if (typingTimeout) {
        clearTimeout(typingTimeout);
    }
    isTyping = false;
}

function goToProfile() {
    window.location.href = '/profile';
} 