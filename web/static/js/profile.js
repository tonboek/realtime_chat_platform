// Profile management JavaScript
let currentUser = null;
let authToken = null;

// Initialize profile page
document.addEventListener('DOMContentLoaded', function() {
    // Get token from localStorage
    authToken = localStorage.getItem('authToken');
    currentUser = localStorage.getItem('username');
    
    if (!authToken || !currentUser) {
        window.location.href = '/';
        return;
    }

    // Load user profile
    loadProfile();
    
    // Add event listeners
    document.getElementById('updateProfileBtn').addEventListener('click', updateProfile);
    document.getElementById('changePasswordBtn').addEventListener('click', changePassword);
    document.getElementById('backToChatBtn').addEventListener('click', goToChat);
    document.getElementById('logoutBtn').addEventListener('click', logout);
    
    // Avatar upload event listeners
    document.getElementById('selectAvatarBtn').addEventListener('click', () => {
        document.getElementById('avatarFile').click();
    });
    document.getElementById('avatarFile').addEventListener('change', handleAvatarFileSelect);
    document.getElementById('uploadAvatarBtn').addEventListener('click', uploadAvatar);
});

// Load user profile data
async function loadProfile() {
    try {
        console.log('Loading profile with token:', authToken ? 'Token exists' : 'No token');
        const response = await fetch('/api/profile/', {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${authToken}`,
                'Content-Type': 'application/json'
            }
        });

        console.log('Profile response status:', response.status);
        if (response.ok) {
            const profile = await response.json();
            console.log('Profile data:', profile);
            populateProfileForm(profile);
        } else {
            const errorData = await response.json();
            console.error('Profile error:', errorData);
            showMessage('Ошибка загрузки профиля: ' + (errorData.error || 'Неизвестная ошибка'), 'error');
        }
    } catch (error) {
        console.error('Error loading profile:', error);
        showMessage('Ошибка загрузки профиля', 'error');
    }
}

// Populate profile form with user data
function populateProfileForm(profile) {
    document.getElementById('username').value = profile.username || '';
    document.getElementById('nickname').value = profile.nickname || '';
    document.getElementById('bio').value = profile.bio || '';
    
    // Update avatar image
    updateAvatarPreview(profile.avatar);
}

// Update avatar preview
function updateAvatarPreview(avatarUrl = null) {
    const avatarImg = document.getElementById('userAvatar');
    
    if (avatarUrl) {
        avatarImg.src = avatarUrl;
    } else {
        avatarImg.src = '/static/images/default-avatar.svg';
    }
}

// Update profile information
async function updateProfile() {
    const nickname = document.getElementById('nickname').value;
    const bio = document.getElementById('bio').value;

    try {
        console.log('Updating profile with token:', authToken ? 'Token exists' : 'No token');
        const response = await fetch('/api/profile/', {
            method: 'PUT',
            headers: {
                'Authorization': `Bearer ${authToken}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                nickname: nickname,
                bio: bio
            })
        });

        console.log('Update profile response status:', response.status);
        if (response.ok) {
            const result = await response.json();
            showMessage('Профиль успешно обновлен', 'success');
            populateProfileForm(result.profile);
        } else {
            const error = await response.json();
            console.error('Update profile error:', error);
            showMessage(error.error || 'Ошибка обновления профиля', 'error');
        }
    } catch (error) {
        console.error('Error updating profile:', error);
        showMessage('Ошибка обновления профиля', 'error');
    }
}

// Change password
async function changePassword() {
    const currentPassword = document.getElementById('currentPassword').value;
    const newPassword = document.getElementById('newPassword').value;

    if (!currentPassword || !newPassword) {
        showMessage('Пожалуйста, заполните все поля', 'error');
        return;
    }

    if (newPassword.length < 6) {
        showMessage('Новый пароль должен содержать минимум 6 символов', 'error');
        return;
    }

    try {
        const response = await fetch('/api/profile/password', {
            method: 'PUT',
            headers: {
                'Authorization': `Bearer ${authToken}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                current_password: currentPassword,
                new_password: newPassword
            })
        });

        if (response.ok) {
            showMessage('Пароль успешно изменен', 'success');
            // Clear password fields
            document.getElementById('currentPassword').value = '';
            document.getElementById('newPassword').value = '';
        } else {
            const error = await response.json();
            showMessage(error.error || 'Ошибка изменения пароля', 'error');
        }
    } catch (error) {
        console.error('Error changing password:', error);
        showMessage('Ошибка изменения пароля', 'error');
    }
}

// Navigate back to chat
function goToChat() {
    // Don't clear localStorage, just navigate back
    window.location.href = '/';
}

// Logout user
function logout() {
    localStorage.removeItem('authToken');
    localStorage.removeItem('username');
    window.location.href = '/';
}

// Show message to user
function showMessage(message, type) {
    const messagesDiv = document.getElementById('messages');
    const alertClass = type === 'success' ? 'alert-success' : 'alert-danger';
    
    const alertDiv = document.createElement('div');
    alertDiv.className = `alert ${alertClass} alert-dismissible fade show`;
    alertDiv.innerHTML = `
        ${message}
        <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
    `;
    
    messagesDiv.appendChild(alertDiv);
    
    // Auto-remove after 5 seconds
    setTimeout(() => {
        if (alertDiv.parentNode) {
            alertDiv.remove();
        }
    }, 5000);
}

// Handle avatar file selection
function handleAvatarFileSelect(event) {
    const file = event.target.files[0];
    if (!file) return;

    // Validate file type
    const validTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/gif', 'image/webp'];
    if (!validTypes.includes(file.type)) {
        showMessage('Неподдерживаемый тип файла. Используйте JPG, PNG, GIF или WebP', 'error');
        return;
    }

    // Validate file size (5MB)
    if (file.size > 5 * 1024 * 1024) {
        showMessage('Файл слишком большой. Максимальный размер: 5MB', 'error');
        return;
    }

    // Show upload button
    document.getElementById('uploadAvatarBtn').style.display = 'inline-block';
    
    // Show preview
    const reader = new FileReader();
    reader.onload = function(e) {
        updateAvatarPreview(e.target.result);
    };
    reader.readAsDataURL(file);
}

// Upload avatar file
async function uploadAvatar() {
    const fileInput = document.getElementById('avatarFile');
    const file = fileInput.files[0];
    
    if (!file) {
        showMessage('Пожалуйста, выберите файл', 'error');
        return;
    }

    const formData = new FormData();
    formData.append('avatar', file);

    try {
        const response = await fetch('/api/profile/avatar', {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${authToken}`
            },
            body: formData
        });

        if (response.ok) {
            const result = await response.json();
            showMessage('Аватар успешно загружен', 'success');
            
            // Update avatar preview with new URL
            updateAvatarPreview(result.avatar_url);
            
            // Hide upload button and clear file input
            document.getElementById('uploadAvatarBtn').style.display = 'none';
            fileInput.value = '';
        } else {
            const error = await response.json();
            showMessage(error.error || 'Ошибка загрузки аватара', 'error');
        }
    } catch (error) {
        console.error('Error uploading avatar:', error);
        showMessage('Ошибка загрузки аватара', 'error');
    }
} 