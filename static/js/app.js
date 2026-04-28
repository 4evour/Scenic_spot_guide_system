document.addEventListener('DOMContentLoaded', function() {
    const chatInput = document.getElementById('chatInput');
    const sendBtn = document.getElementById('sendBtn');
    const voiceBtn = document.getElementById('voiceBtn');
    const speakerBtn = document.getElementById('speakerBtn');
    const chatMessages = document.getElementById('chatMessages');
    const uploadArea = document.getElementById('uploadArea');

    let token = localStorage.getItem('token');
    let currentUser = null;
    let isSpeaking = false;
    let isListening = false;
    let recognition = null;
    let currentAudio = null;

    if (token) {
        loadUserInfo();
    }

    function loadUserInfo() {
        fetch('/api/v1/user/me', {
            headers: {
                'Authorization': 'Bearer ' + token
            }
        })
        .then(res => res.json())
        .then(data => {
            if (data.code === 0) {
                currentUser = data.data;
                document.getElementById('loginBtn').classList.add('hidden');
                document.getElementById('userInfo').classList.remove('hidden');
                document.getElementById('currentUser').textContent = currentUser.username;
                document.getElementById('userRole').textContent = currentUser.role === 'admin' ? '管理员' : '游客';
            }
        })
        .catch(err => {
            localStorage.removeItem('token');
        });
    }

    function openModal(type) {
        document.getElementById('loginModal').classList.add('show');
        switchTab(type);
    }

    function closeModal() {
        document.getElementById('loginModal').classList.remove('show');
    }

    function switchTab(tab) {
        document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
        document.querySelectorAll('.form-container').forEach(form => form.classList.add('hidden'));

        document.querySelector(`.tab-btn:nth-child(${tab === 'login' ? 1 : 2})`).classList.add('active');
        document.getElementById(`${tab}Form`).classList.remove('hidden');
    }

    function login() {
        const username = document.getElementById('loginUsername').value;
        const password = document.getElementById('loginPassword').value;

        if (!username || !password) {
            alert('请填写用户名和密码');
            return;
        }

        fetch('/api/v1/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ username, password })
        })
        .then(res => res.json())
        .then(data => {
            if (data.code === 0) {
                token = data.data.token;
                localStorage.setItem('token', token);
                currentUser = {
                    username: data.data.username,
                    role: data.data.role
                };
                closeModal();
                document.getElementById('loginBtn').classList.add('hidden');
                document.getElementById('userInfo').classList.remove('hidden');
                document.getElementById('currentUser').textContent = currentUser.username;
                document.getElementById('userRole').textContent = currentUser.role === 'admin' ? '管理员' : '游客';
                showAvatarEmotion('happy');
                speakText('登录成功，欢迎回来！');
                alert('登录成功');
            } else {
                alert(data.message);
            }
        })
        .catch(err => {
            alert('登录失败，请重试');
        });
    }

    function register() {
        const username = document.getElementById('regUsername').value;
        const password = document.getElementById('regPassword').value;
        const email = document.getElementById('regEmail').value;
        const role = document.getElementById('regRole').value;

        if (!username || !password) {
            alert('请填写用户名和密码');
            return;
        }

        fetch('/api/v1/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ username, password, email, role })
        })
        .then(res => res.json())
        .then(data => {
            if (data.code === 0) {
                alert('注册成功，请登录');
                switchTab('login');
            } else {
                alert(data.message);
            }
        })
        .catch(err => {
            alert('注册失败，请重试');
        });
    }

    function logout() {
        localStorage.removeItem('token');
        token = null;
        currentUser = null;
        document.getElementById('loginBtn').classList.remove('hidden');
        document.getElementById('userInfo').classList.add('hidden');
        stopSpeaking();
        alert('已退出登录');
    }

    function showAvatarEmotion(emotion) {
        const avatar = document.getElementById('digitalAvatar');
        const emotionTag = document.getElementById('emotionTag');

        avatar.classList.remove('speaking', 'listening', 'happy');

        if (emotion) {
            avatar.classList.add(emotion);

            const emotionTexts = {
                'happy': '表情：开心',
                'speaking': '表情：说话中',
                'listening': '表情：倾听'
            };

            if (emotionTag && emotionTexts[emotion]) {
                emotionTag.textContent = emotionTexts[emotion];
                emotionTag.className = 'status-tag emotion-tag ' + emotion;
            }
        }
    }

    function startSpeaking() {
        const avatar = document.getElementById('digitalAvatar');
        const mouth = document.getElementById('mouth');
        const speakingIndicator = document.getElementById('speakingIndicator');
        const voiceStatus = document.querySelector('.voice-state');

        isSpeaking = true;
        avatar.classList.add('speaking');
        if (mouth) mouth.classList.add('speaking');
        if (speakingIndicator) speakingIndicator.classList.add('active');
        if (voiceStatus) voiceStatus.textContent = 'AI正在说话...';
    }

    function stopSpeaking() {
        const avatar = document.getElementById('digitalAvatar');
        const mouth = document.getElementById('mouth');
        const speakingIndicator = document.getElementById('speakingIndicator');
        const voiceStatus = document.querySelector('.voice-state');

        isSpeaking = false;
        avatar.classList.remove('speaking');
        if (mouth) mouth.classList.remove('speaking');
        if (speakingIndicator) speakingIndicator.classList.remove('active');
        if (voiceStatus) voiceStatus.textContent = '等待输入...';

        if (currentAudio) {
            currentAudio.pause();
            currentAudio = null;
        }
    }

    function speakText(text, useWebSpeech = true) {
        stopSpeaking();

        if (useWebSpeech && ('speechSynthesis' in window)) {
            const utterance = new SpeechSynthesisUtterance(text);
            utterance.lang = 'zh-CN';
            utterance.rate = 1.0;
            utterance.pitch = 1.0;

            const voices = speechSynthesis.getVoices();
            const zhVoice = voices.find(v => v.lang.includes('zh'));
            if (zhVoice) {
                utterance.voice = zhVoice;
            }

            utterance.onstart = () => {
                startSpeaking();
            };

            utterance.onend = () => {
                stopSpeaking();
                showAvatarEmotion('happy');
            };

            utterance.onerror = () => {
                stopSpeaking();
            };

            speechSynthesis.speak(utterance);
        } else {
            fetchTTSAudio(text);
        }
    }

    function fetchTTSAudio(text) {
        fetch('/api/v1/ai/tts', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                text: text,
                voice: 'female_tianmei',
                speed: '1.0'
            })
        })
        .then(res => res.json())
        .then(data => {
            if (data.code === 0 && data.data.audio_url) {
                playAudio(data.data.audio_url);
            } else {
                console.warn('TTS API unavailable, using Web Speech API');
                speakText(text, true);
            }
        })
        .catch(err => {
            console.warn('TTS API error, falling back to Web Speech API');
            speakText(text, true);
        });
    }

    function playAudio(url) {
        stopSpeaking();
        startSpeaking();

        currentAudio = new Audio(url);
        currentAudio.play();

        currentAudio.onended = () => {
            stopSpeaking();
            showAvatarEmotion('happy');
        };

        currentAudio.onerror = () => {
            stopSpeaking();
        };
    }

    function initVoiceInput() {
        if (!('webkitSpeechRecognition' in window) && !('SpeechRecognition' in window)) {
            console.warn('Speech recognition not supported');
            return;
        }

        const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
        recognition = new SpeechRecognition();
        recognition.lang = 'zh-CN';
        recognition.continuous = false;
        recognition.interimResults = false;

        recognition.onstart = () => {
            isListening = true;
            showAvatarEmotion('listening');
            voiceBtn.classList.add('listening');
            document.querySelector('.voice-state').textContent = '正在聆听...';
        };

        recognition.onresult = (event) => {
            const transcript = event.results[0][0].transcript;
            chatInput.value = transcript;
            sendMessage(transcript);
        };

        recognition.onerror = (event) => {
            console.error('Speech recognition error:', event.error);
            stopListening();
            if (event.error !== 'no-speech') {
                addMessage('语音识别出错，请重试', true);
            }
        };

        recognition.onend = () => {
            stopListening();
        };
    }

    function startListening() {
        if (recognition) {
            try {
                recognition.start();
            } catch (e) {
                console.error('Recognition start error:', e);
            }
        }
    }

    function stopListening() {
        isListening = false;
        voiceBtn.classList.remove('listening');
        showAvatarEmotion(null);
        document.querySelector('.voice-state').textContent = '等待输入...';
        if (recognition) {
            try {
                recognition.stop();
            } catch (e) {
            }
        }
    }

    function addMessage(text, isBot = true) {
        const messageDiv = document.createElement('div');
        messageDiv.className = `message ${isBot ? 'bot-message' : 'user-message'}`;

        messageDiv.innerHTML = `
            <div class="message-avatar ${isBot ? 'bot' : 'user'}"></div>
            <div class="message-content">
                <p>${text}</p>
            </div>
        `;

        chatMessages.appendChild(messageDiv);
        chatMessages.scrollTop = chatMessages.scrollHeight;
    }

    async function sendMessage(text) {
        if (!text.trim()) return;

        addMessage(text, false);
        chatInput.value = '';

        showAvatarEmotion('listening');
        document.querySelector('.voice-state').textContent = 'AI正在思考...';

        try {
            const headers = {
                'Content-Type': 'application/json'
            };
            if (token) {
                headers['Authorization'] = 'Bearer ' + token;
            }

            const response = await fetch('/api/v1/ai/chat', {
                method: 'POST',
                headers: headers,
                body: JSON.stringify({ message: text }),
            });

            const data = await response.json();

            chatMessages.removeChild(chatMessages.lastChild);

            if (data.code === 0) {
                const aiResponse = data.data.response;
                addMessage(aiResponse, true);
                speakText(aiResponse);
            } else {
                addMessage('抱歉，我暂时无法回答您的问题。', true);
                document.querySelector('.voice-state').textContent = '服务暂时不可用';
            }
        } catch (error) {
            chatMessages.removeChild(chatMessages.lastChild);
            addMessage('网络错误，请稍后重试。', true);
            document.querySelector('.voice-state').textContent = '网络错误';
            console.error('API调用失败:', error);
        }
    }

    sendBtn.addEventListener('click', function() {
        sendMessage(chatInput.value);
    });

    chatInput.addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            sendMessage(chatInput.value);
        }
    });

    voiceBtn.addEventListener('click', function() {
        if (isListening) {
            stopListening();
        } else {
            if (!recognition) {
                initVoiceInput();
            }
            startListening();
        }
    });

    speakerBtn.addEventListener('click', function() {
        const messages = chatMessages.querySelectorAll('.bot-message');
        if (messages.length > 0) {
            const lastBotMessage = messages[messages.length - 1];
            const text = lastBotMessage.querySelector('p').textContent;
            speakText(text);
        } else {
            speakText('目前没有可播放的回复内容。');
        }
    });

    if ('speechSynthesis' in window) {
        speechSynthesis.getVoices();
        speechSynthesis.onvoiceschanged = () => {
            speechSynthesis.getVoices();
        };
    }

    initVoiceInput();

    uploadArea.addEventListener('dragover', function(e) {
        e.preventDefault();
        uploadArea.style.borderColor = 'rgba(79, 172, 254, 0.5)';
        uploadArea.style.background = 'rgba(79, 172, 254, 0.05)';
    });

    uploadArea.addEventListener('dragleave', function() {
        uploadArea.style.borderColor = 'rgba(255, 255, 255, 0.2)';
        uploadArea.style.background = 'transparent';
    });

    uploadArea.addEventListener('drop', function(e) {
        e.preventDefault();
        uploadArea.style.borderColor = 'rgba(255, 255, 255, 0.2)';
        uploadArea.style.background = 'transparent';

        const files = e.dataTransfer.files;
        if (files.length > 0) {
            addMessage(`正在处理文件: ${files[0].name}...`, true);
        }
    });

    uploadArea.addEventListener('click', function() {
        addMessage('文件上传功能开发中...', true);
    });

    document.querySelectorAll('.btn-preview, .btn-publish, .btn-view').forEach(btn => {
        btn.addEventListener('click', function() {
            const action = this.textContent;
            addMessage(`${action}功能开发中...`, true);
        });
    });

    document.querySelectorAll('input, select, textarea').forEach(input => {
        input.addEventListener('change', function() {
            console.log('设置已更新:', this.name || this.parentElement.querySelector('label').textContent);
        });
    });

    document.getElementById('loginModal').addEventListener('click', function(e) {
        if (e.target === this) {
            closeModal();
        }
    });

    window.openModal = openModal;
    window.closeModal = closeModal;
    window.switchTab = switchTab;
    window.login = login;
    window.register = register;
    window.logout = logout;
    window.sendMessage = sendMessage;
});
