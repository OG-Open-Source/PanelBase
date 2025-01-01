#!/bin/bash

# 初始化配置文件
CONFIG_DIR="/opt/panelbase/config"
CONFIG_FILE="$CONFIG_DIR/user.conf"
SESSION_FILE="$CONFIG_DIR/sessions.conf"
SECURITY_CONF="$CONFIG_DIR/security.conf"

# 如果 security.conf 不存在，創建默認配置
if [ ! -f "$SECURITY_CONF" ]; then
	cat >"$SECURITY_CONF" <<EOF
# 安全配置
SECURITY_HEADERS_CSP="default-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdnjs.cloudflare.com https://raw.githubusercontent.com; img-src 'self' data: https:; font-src 'self' https://cdnjs.cloudflare.com; connect-src 'self'"
SESSION_LIFETIME=86400
WHITELIST_FILES="*.html *.css *.js *.png *.jpg *.jpeg *.gif *.svg *.ico *.woff *.woff2 *.ttf *.eot"
BLACKLIST_FILES=""
ACCESS_CONTROL_MODE="whitelist"
ALLOW_HTML_REFERENCE="true"
CACHE_MAX_AGE=3600
LOG_FILE="/opt/panelbase/logs/auth.log"
EOF
	chmod 600 "$SECURITY_CONF"
fi

# 載入安全配置
source "$SECURITY_CONF"

# 確保日誌目錄存在
mkdir -p "$(dirname "$LOG_FILE")"
touch "$LOG_FILE"
chmod 600 "$LOG_FILE"

log_auth_event() {
	local level="$1"
	local message="$2"
	echo "[$(date '+%Y-%m-%d %H:%M:%S')] [$level] $message" >>"$LOG_FILE"
}

WHITELIST_REGEX=$(echo "$WHITELIST_FILES" | sed 's/\./\\./g' | sed 's/\*/.*/g' | tr ' ' '|')
BLACKLIST_REGEX=$(echo "$BLACKLIST_FILES" | sed 's/\./\\./g' | sed 's/\*/.*/g' | tr ' ' '|')

check_file_access() {
	local file="$1"
	local referer="$2"
	local filename=$(basename "$file")
	local is_allowed=false

	case "$ACCESS_CONTROL_MODE" in
	"whitelist")
		if echo "$filename" | grep -qE "^($WHITELIST_REGEX)$"; then
			is_allowed=true
		elif [ "$ALLOW_HTML_REFERENCE" = "true" ] && [ -n "$referer" ] && echo "$referer" | grep -q "^/.*\.html"; then
			is_allowed=true
		fi
		;;
	"blacklist")
		if ! echo "$filename" | grep -qE "^($BLACKLIST_REGEX)$"; then
			is_allowed=true
		elif [ "$ALLOW_HTML_REFERENCE" = "true" ] && [ -n "$referer" ] && echo "$referer" | grep -q "^/.*\.html"; then
			is_allowed=true
		fi
		;;
	*)
		log_auth_event "ERROR" "Invalid ACCESS_CONTROL_MODE: $ACCESS_CONTROL_MODE"
		is_allowed=false
		;;
	esac

	[ "$is_allowed" = "true" ]
}

SECURITY_HEADERS() {
	local content_type="${1:-text/html}"
	local status="$2"

	echo "Content-type: $content_type"
	echo "X-Content-Type-Options: nosniff"
	echo "X-Frame-Options: SAMEORIGIN"
	echo "X-XSS-Protection: 1; mode=block"
	echo "Referrer-Policy: strict-origin-when-cross-origin"
	echo "Permissions-Policy: geolocation=(), microphone=(), camera=()"
	echo "Content-Security-Policy: $SECURITY_HEADERS_CSP"
	[ -n "$status" ] && echo "Status: $status"
	echo
}

SHOW_ERROR() {
	local status="$1"
	local code="$2"
	local message="$3"

	log_auth_event "WARN" "$message"

	if [ -n "$IS_CURL" ]; then
		SECURITY_HEADERS "application/json" "$status"
		echo "{\"status\":\"error\",\"code\":\"$status\",\"message\":\"$message\"}"
	else
		SECURITY_HEADERS "text/html" "$status"
		case "$code" in
		"403")
			cat <<'ERROR_403_HTML'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>403 Forbidden - PanelBase</title>
    <link rel="icon" type="image/png" sizes="128x128" href="/favicon.png">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css" />
    <style>
        :root {
            --primary-color: #007AFF;
            --primary-gradient: linear-gradient(135deg, #0A84FF, #0066CC);
            --error-color: #FF3B30;
            --background-color: #F2F2F7;
            --text-color: #1C1C1E;
            --secondary-text: #8E8E93;
            --border-radius: 12px;
            --border-radius-lg: 16px;
            --border-radius-sm: 8px;
            --container-bg: rgba(255, 255, 255, 0.95);
            --container-shadow: 0 10px 30px rgba(0, 0, 0, 0.08);
            --body-gradient: linear-gradient(135deg, #F6F8FB, #E9EEF3);
            --primary-rgb: 0, 122, 255;
        }
        
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
        }
        
        body {
            background: var(--body-gradient);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
            position: relative;
            overflow: hidden;
            isolation: isolate;
        }
        
        body::before {
            content: '';
            position: absolute;
            width: 150%;
            height: 150%;
            background: radial-gradient(circle, rgba(255, 255, 255, 0.8) 0%, rgba(255, 255, 255, 0) 70%);
            animation: rotate 20s linear infinite;
            z-index: 0;
        }
        
        body::after {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: linear-gradient(45deg, transparent 45%, rgba(var(--primary-rgb), 0.03) 50%, transparent 55%), linear-gradient(-45deg, transparent 45%, rgba(var(--primary-rgb), 0.03) 50%, transparent 55%);
            background-size: 30px 30px;
            animation: backgroundMove 60s linear infinite;
            z-index: 0;
        }
        
        .container {
            width: 100%;
            max-width: 380px;
            padding: 32px;
            background: var(--container-bg);
            border-radius: var(--border-radius-lg);
            box-shadow: var(--container-shadow);
            backdrop-filter: blur(16px);
            -webkit-backdrop-filter: blur(16px);
            position: relative;
            z-index: 1;
            text-align: center;
        }
        
        .error-code {
            font-size: 96px;
            font-weight: 700;
            background: var(--primary-gradient);
            background-clip: text;
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 12px;
            letter-spacing: -0.5px;
            line-height: 1;
        }
        
        .error-message {
            font-size: 24px;
            color: var(--text-color);
            margin-bottom: 24px;
            font-weight: 600;
        }
        
        .error-description {
            color: var(--secondary-text);
            margin-bottom: 32px;
            line-height: 1.5;
        }
        
        .back-button {
            display: inline-block;
            padding: 12px 24px;
            background: var(--primary-gradient);
            color: white;
            text-decoration: none;
            border-radius: var(--border-radius);
            font-weight: 600;
            transition: all 0.3s ease;
            position: relative;
            overflow: hidden;
        }
        
        .back-button::before {
            content: '';
            position: absolute;
            top: 0;
            left: -100%;
            width: 200%;
            height: 100%;
            background: linear-gradient(120deg, transparent 0%, transparent 25%, rgba(255, 255, 255, 0.3) 45%, rgba(255, 255, 255, 0.7) 50%, rgba(255, 255, 255, 0.3) 55%, transparent 75%, transparent 100%);
            transform: skewX(-25deg);
            animation: shine 8s infinite;
        }
        
        .back-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(var(--primary-rgb), 0.3);
        }
        
        .back-button:active {
            transform: translateY(0);
        }
        
        @keyframes shine {
            0% {
                left: -200%;
            }
            20%,
            100% {
                left: 200%;
            }
        }
        
        @keyframes rotate {
            from {
                transform: rotate(0deg);
            }
            to {
                transform: rotate(360deg);
            }
        }
        
        @keyframes backgroundMove {
            from {
                background-position: 0 0;
            }
            to {
                background-position: 100% 100%;
            }
        }
        
        .background-shapes {
            position: absolute;
            width: 100%;
            height: 100%;
            pointer-events: none;
        }
        
        .shape {
            position: absolute;
            background: linear-gradient(135deg, rgba(var(--primary-rgb), 0.2), transparent);
            backdrop-filter: blur(8px);
        }
        
        .shape-1 {
            width: 120px;
            height: 120px;
            left: -60px;
            top: 50%;
            transform: rotate(45deg) translateY(-50%);
            animation: shapeFloat1 12s ease-in-out infinite;
        }
        
        .shape-2 {
            width: 80px;
            height: 80px;
            right: -40px;
            top: 30%;
            border-radius: 50%;
            animation: shapeFloat2 15s ease-in-out infinite;
        }
        
        .shape-3 {
            width: 60px;
            height: 60px;
            left: 15%;
            bottom: 10%;
            border-radius: 30% 70% 70% 30% / 30% 30% 70% 70%;
            animation: shapeFloat3 10s ease-in-out infinite, shapeRotate 20s linear infinite;
        }
        
        .shape-4 {
            width: 40px;
            height: 40px;
            right: 15%;
            top: 15%;
            clip-path: polygon(50% 0%, 100% 50%, 50% 100%, 0% 50%);
            animation: shapeFloat4 8s ease-in-out infinite;
        }
        
        @keyframes shapeFloat1 {
            0%,
            100% {
                transform: rotate(45deg) translateY(-50%) translateX(0);
            }
            50% {
                transform: rotate(45deg) translateY(-50%) translateX(30px);
            }
        }
        
        @keyframes shapeFloat2 {
            0%,
            100% {
                transform: translateY(0) scale(1);
            }
            50% {
                transform: translateY(-30px) scale(1.1);
            }
        }
        
        @keyframes shapeFloat3 {
            0%,
            100% {
                transform: translate(0, 0);
            }
            50% {
                transform: translate(20px, -20px);
            }
        }
        
        @keyframes shapeFloat4 {
            0%,
            100% {
                transform: translate(0, 0) rotate(0deg);
            }
            50% {
                transform: translate(-15px, 15px) rotate(180deg);
            }
        }
        
        @keyframes shapeRotate {
            from {
                transform: rotate(0deg);
            }
            to {
                transform: rotate(360deg);
            }
        }
        
        :root.dark-mode {
            --background-color: #000000;
            --text-color: #FFFFFF;
            --secondary-text: #8E8E93;
            --container-bg: rgba(28, 28, 30, 0.95);
            --container-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
            --body-gradient: linear-gradient(135deg, #1C1C1E, #2C2C2E);
            --primary-rgb: 10, 132, 255;
        }
        
        :root.dark-mode body {
            background: var(--body-gradient);
        }
        
        :root.dark-mode .container {
            background: var(--container-bg);
            color: var(--text-color);
        }
        
        :root.dark-mode body::before {
            background: radial-gradient(circle, rgba(255, 255, 255, 0.1) 0%, rgba(255, 255, 255, 0) 70%);
        }
        
        @media (prefers-color-scheme: dark) {
            :root:not(.light-mode) {
                --background-color: #000000;
                --text-color: #FFFFFF;
                --secondary-text: #8E8E93;
                --container-bg: rgba(28, 28, 30, 0.95);
                --container-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
                --body-gradient: linear-gradient(135deg, #1C1C1E, #2C2C2E);
                --primary-rgb: 10, 132, 255;
            }
            :root:not(.light-mode) body {
                background: var(--body-gradient);
            }
            :root:not(.light-mode) .container {
                background: var(--container-bg);
                color: var(--text-color);
            }
        }
        
        @media (max-width: 480px) {
            body {
                padding: 0;
                background: var(--container-bg);
            }
            .container {
                max-width: 100%;
                margin: 0;
                padding: 20px;
                min-height: 100vh;
                border-radius: 0;
                box-shadow: none;
            }
            .background-shapes {
                display: none;
            }
            body::before,
            body::after {
                display: none;
            }
        }
        
        .footer {
            position: fixed;
            bottom: 20px;
            left: 50%;
            transform: translateX(-50%);
            color: var(--secondary-text);
            font-size: 12px;
            opacity: 0.9;
            transition: all 0.3s ease;
            white-space: nowrap;
            background: var(--container-bg);
            padding: 8px 16px;
            border-radius: var(--border-radius);
            box-shadow: var(--container-shadow);
            backdrop-filter: blur(8px);
            z-index: 2;
        }
        
        .footer:hover {
            opacity: 1;
            transform: translateX(-50%) translateY(-2px);
        }
        
        .footer a {
            color: var(--primary-color);
            text-decoration: none;
            font-weight: 500;
            transition: all 0.3s ease;
        }
        
        .footer a:hover {
            text-decoration: underline;
            opacity: 0.9;
        }
        
        .language-switcher {
            margin-top: 24px;
            text-align: center;
            display: flex;
            justify-content: center;
            gap: 8px;
        }
        
        .language-switcher button {
            background: none;
            border: 2px solid transparent;
            padding: 6px 12px;
            color: var(--secondary-text);
            font-size: 14px;
            cursor: pointer;
            transition: all 0.2s ease;
            border-radius: var(--border-radius-sm);
        }
        
        .language-switcher button:hover {
            background: rgba(0, 122, 255, 0.1);
            color: var(--primary-color);
        }
        
        .language-switcher button[data-active="true"] {
            border-color: var(--primary-color);
            color: var(--primary-color);
        }
        
        .theme-switcher {
            position: absolute;
            top: 20px;
            right: 20px;
            z-index: 2;
        }
        
        .theme-switcher button {
            background: none;
            border: none;
            color: var(--secondary-text);
            font-size: 18px;
            cursor: pointer;
            padding: 8px;
            border-radius: var(--border-radius-sm);
            transition: all 0.3s ease;
            width: 36px;
            height: 36px;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        
        .theme-switcher button i {
            width: 18px;
            text-align: center;
        }
        
        .theme-switcher button:hover {
            background: rgba(0, 122, 255, 0.1);
            color: var(--primary-color);
        }
    </style>
    <script>
        const loginForm = document.getElementById('login-form');
        const errorMessage = document.getElementById('error-message');
        const usernameInput = document.getElementById('username');
        const passwordInput = document.getElementById('password');
        const rememberMeCheckbox = document.getElementById('remember-me');
        const loginButton = document.getElementById('login-button');
        const languageSwitcher = document.querySelector('.language-switcher');
        const themeSwitcher = document.querySelector('.theme-switcher');
        const themeSwitcherButton = themeSwitcher.querySelector('button');
        const themeSwitcherIcon = themeSwitcherButton.querySelector('i');
        const footer = document.querySelector('.footer');
        const backgroundShapes = document.querySelector('.background-shapes');
        const shapes = document.querySelectorAll('.shape');
        
        const translations = {
            en: {
                title: 'PanelBase Login',
                username: 'Username',
                password: 'Password',
                rememberMe: 'Remember me',
                login: 'Login',
                errorMessage: 'Invalid username or password',
                footerText: 'Powered by PanelBase',
                lightMode: 'Light Mode',
                darkMode: 'Dark Mode',
            },
            zh: {
                title: 'PanelBase 登录',
                username: '用户名',
                password: '密码',
                rememberMe: '记住我',
                login: '登录',
                errorMessage: '无效的用户名或密码',
                footerText: '由 PanelBase 提供支持',
                lightMode: '浅色模式',
                darkMode: '深色模式',
            },
        };
        
        let currentLanguage = 'en';
        let isDarkMode = false;
        
        function updateLanguage() {
            const translation = translations[currentLanguage];
            document.title = translation.title;
            document.querySelector('h1').textContent = translation.title;
            usernameInput.placeholder = translation.username;
            passwordInput.placeholder = translation.password;
            rememberMeCheckbox.nextElementSibling.textContent = translation.rememberMe;
            loginButton.textContent = translation.login;
            errorMessage.textContent = translation.errorMessage;
            footer.innerHTML = translation.footerText;
            themeSwitcherButton.setAttribute('title', translation.lightMode);
            themeSwitcherIcon.classList.remove('fa-sun', 'fa-moon');
            themeSwitcherIcon.classList.add(isDarkMode ? 'fa-moon' : 'fa-sun');
            themeSwitcherButton.setAttribute('title', isDarkMode ? translation.darkMode : translation.lightMode);
            
            languageSwitcher.querySelectorAll('button').forEach(button => {
                button.dataset.active = button.dataset.lang === currentLanguage;
            });
        }
        
        function toggleDarkMode() {
            isDarkMode = !isDarkMode;
            document.documentElement.classList.toggle('dark-mode', isDarkMode);
            themeSwitcherIcon.classList.toggle('fa-sun', !isDarkMode);
            themeSwitcherIcon.classList.toggle('fa-moon', isDarkMode);
            themeSwitcherButton.setAttribute('title', isDarkMode ? translations[currentLanguage].darkMode : translations[currentLanguage].lightMode);
            localStorage.setItem('darkMode', isDarkMode);
        }
        
        function handleLogin(event) {
            event.preventDefault();
            const username = usernameInput.value;
            const password = passwordInput.value;
            const rememberMe = rememberMeCheckbox.checked;
            
            // Perform login logic here
            // If login successful, redirect to the desired page
            // If login fails, show error message
        }
        
        function handleLanguageSwitch(event) {
            currentLanguage = event.target.dataset.lang;
            updateLanguage();
        }
        
        function handleThemeSwitch() {
            toggleDarkMode();
        }
        
        function init() {
            isDarkMode = localStorage.getItem('darkMode') === 'true';
            toggleDarkMode();
            updateLanguage();
            loginForm.addEventListener('submit', handleLogin);
            languageSwitcher.addEventListener('click', handleLanguageSwitch);
            themeSwitcherButton.addEventListener('click', handleThemeSwitch);
        }
        
        init();
    </script>
</head>
<body>
    <div class="background-shapes">
        <div class="shape shape-1"></div>
        <div class="shape shape-2"></div>
        <div class="shape shape-3"></div>
        <div class="shape shape-4"></div>
    </div>
    <div class="container">
        <div class="theme-switcher">
            <button onclick="toggleTheme()" id="themeToggle">
                <i class="fas fa-moon"></i>
            </button>
        </div>
        <div class="error-code">403</div>
        <h1 class="error-message" data-i18n="error-title">Access Forbidden</h1>
        <p class="error-description" data-i18n="error-description">
            You don't have permission to access this resource. Please check your credentials or contact the administrator.
        </p>
        <a href="/s/login" class="back-button" data-i18n="back-button">Back to Login</a>
        <div class="language-switcher">
            <button onclick="changeLanguage('en')" data-lang="en">EN</button>
            <button onclick="changeLanguage('zh-TW')" data-lang="zh-TW">繁</button>
            <button onclick="changeLanguage('zh-CN')" data-lang="zh-CN">简</button>
        </div>
    </div>

    <div class="footer">
        Powered by <a href="https://github.com/OG-Open-Source" target="_blank">OG Open Source</a>
    </div>

    <script>
        (function() {
            const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
            const savedTheme = localStorage.getItem('theme-preference');

            if (savedTheme) {
                document.documentElement.classList.add(`${savedTheme}-mode`);
            } else if (prefersDark) {
                document.documentElement.classList.add('dark-mode');
            }
        })();
    </script>
</body>
</html>
ERROR_403_HTML
			;;
		"404")
			cat <<'ERROR_404_HTML'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>404 Not Found - PanelBase</title>
    <link rel="icon" type="image/png" sizes="128x128" href="/favicon.png">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css" />
    <style>
        :root {
            --primary-color: #007AFF;
            --primary-gradient: linear-gradient(135deg, #0A84FF, #0066CC);
            --error-color: #FF3B30;
            --background-color: #F2F2F7;
            --text-color: #1C1C1E;
            --secondary-text: #8E8E93;
            --border-radius: 12px;
            --border-radius-lg: 16px;
            --border-radius-sm: 8px;
            --container-bg: rgba(255, 255, 255, 0.95);
            --container-shadow: 0 10px 30px rgba(0, 0, 0, 0.08);
            --body-gradient: linear-gradient(135deg, #F6F8FB, #E9EEF3);
            --primary-rgb: 0, 122, 255;
        }
        
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
        }
        
        body {
            background: var(--body-gradient);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
            position: relative;
            overflow: hidden;
            isolation: isolate;
        }
        
        body::before {
            content: '';
            position: absolute;
            width: 150%;
            height: 150%;
            background: radial-gradient(circle, rgba(255, 255, 255, 0.8) 0%, rgba(255, 255, 255, 0) 70%);
            animation: rotate 20s linear infinite;
            z-index: 0;
        }
        
        body::after {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: linear-gradient(45deg, transparent 45%, rgba(var(--primary-rgb), 0.03) 50%, transparent 55%), linear-gradient(-45deg, transparent 45%, rgba(var(--primary-rgb), 0.03) 50%, transparent 55%);
            background-size: 30px 30px;
            animation: backgroundMove 60s linear infinite;
            z-index: 0;
        }
        
        .container {
            width: 100%;
            max-width: 380px;
            padding: 32px;
            background: var(--container-bg);
            border-radius: var(--border-radius-lg);
            box-shadow: var(--container-shadow);
            backdrop-filter: blur(16px);
            -webkit-backdrop-filter: blur(16px);
            position: relative;
            z-index: 1;
            text-align: center;
        }
        
        .error-code {
            font-size: 96px;
            font-weight: 700;
            background: var(--primary-gradient);
            background-clip: text;
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 12px;
            letter-spacing: -0.5px;
            line-height: 1;
        }
        
        .error-message {
            font-size: 24px;
            color: var(--text-color);
            margin-bottom: 24px;
            font-weight: 600;
        }
        
        .error-description {
            color: var(--secondary-text);
            margin-bottom: 32px;
            line-height: 1.5;
        }
        
        .back-button {
            display: inline-block;
            padding: 12px 24px;
            background: var(--primary-gradient);
            color: white;
            text-decoration: none;
            border-radius: var(--border-radius);
            font-weight: 600;
            transition: all 0.3s ease;
            position: relative;
            overflow: hidden;
        }
        
        .back-button::before {
            content: '';
            position: absolute;
            top: 0;
            left: -100%;
            width: 200%;
            height: 100%;
            background: linear-gradient(120deg, transparent 0%, transparent 25%, rgba(255, 255, 255, 0.3) 45%, rgba(255, 255, 255, 0.7) 50%, rgba(255, 255, 255, 0.3) 55%, transparent 75%, transparent 100%);
            transform: skewX(-25deg);
            animation: shine 8s infinite;
        }
        
        .back-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(var(--primary-rgb), 0.3);
        }
        
        .back-button:active {
            transform: translateY(0);
        }
        
        @keyframes shine {
            0% {
                left: -200%;
            }
            20%,
            100% {
                left: 200%;
            }
        }
        
        @keyframes rotate {
            from {
                transform: rotate(0deg);
            }
            to {
                transform: rotate(360deg);
            }
        }
        
        @keyframes backgroundMove {
            from {
                background-position: 0 0;
            }
            to {
                background-position: 100% 100%;
            }
        }
        
        .background-shapes {
            position: absolute;
            width: 100%;
            height: 100%;
            pointer-events: none;
        }
        
        .shape {
            position: absolute;
            background: linear-gradient(135deg, rgba(var(--primary-rgb), 0.2), transparent);
            backdrop-filter: blur(8px);
        }
        
        .shape-1 {
            width: 120px;
            height: 120px;
            left: -60px;
            top: 50%;
            transform: rotate(45deg) translateY(-50%);
            animation: shapeFloat1 12s ease-in-out infinite;
        }
        
        .shape-2 {
            width: 80px;
            height: 80px;
            right: -40px;
            top: 30%;
            border-radius: 50%;
            animation: shapeFloat2 15s ease-in-out infinite;
        }
        
        .shape-3 {
            width: 60px;
            height: 60px;
            left: 15%;
            bottom: 10%;
            border-radius: 30% 70% 70% 30% / 30% 30% 70% 70%;
            animation: shapeFloat3 10s ease-in-out infinite, shapeRotate 20s linear infinite;
        }
        
        .shape-4 {
            width: 40px;
            height: 40px;
            right: 15%;
            top: 15%;
            clip-path: polygon(50% 0%, 100% 50%, 50% 100%, 0% 50%);
            animation: shapeFloat4 8s ease-in-out infinite;
        }
        
        @keyframes shapeFloat1 {
            0%,
            100% {
                transform: rotate(45deg) translateY(-50%) translateX(0);
            }
            50% {
                transform: rotate(45deg) translateY(-50%) translateX(30px);
            }
        }
        
        @keyframes shapeFloat2 {
            0%,
            100% {
                transform: translateY(0) scale(1);
            }
            50% {
                transform: translateY(-30px) scale(1.1);
            }
        }
        
        @keyframes shapeFloat3 {
            0%,
            100% {
                transform: translate(0, 0);
            }
            50% {
                transform: translate(20px, -20px);
            }
        }
        
        @keyframes shapeFloat4 {
            0%,
            100% {
                transform: translate(0, 0) rotate(0deg);
            }
            50% {
                transform: translate(-15px, 15px) rotate(180deg);
            }
        }
        
        @keyframes shapeRotate {
            from {
                transform: rotate(0deg);
            }
            to {
                transform: rotate(360deg);
            }
        }
        
        :root.dark-mode {
            --background-color: #000000;
            --text-color: #FFFFFF;
            --secondary-text: #8E8E93;
            --container-bg: rgba(28, 28, 30, 0.95);
            --container-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
            --body-gradient: linear-gradient(135deg, #1C1C1E, #2C2C2E);
            --primary-rgb: 10, 132, 255;
        }
        
        :root.dark-mode body {
            background: var(--body-gradient);
        }
        
        :root.dark-mode .container {
            background: var(--container-bg);
            color: var(--text-color);
        }
        
        :root.dark-mode body::before {
            background: radial-gradient(circle, rgba(255, 255, 255, 0.1) 0%, rgba(255, 255, 255, 0) 70%);
        }
        
        @media (prefers-color-scheme: dark) {
            :root:not(.light-mode) {
                --background-color: #000000;
                --text-color: #FFFFFF;
                --secondary-text: #8E8E93;
                --container-bg: rgba(28, 28, 30, 0.95);
                --container-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
                --body-gradient: linear-gradient(135deg, #1C1C1E, #2C2C2E);
                --primary-rgb: 10, 132, 255;
            }
            :root:not(.light-mode) body {
                background: var(--body-gradient);
            }
            :root:not(.light-mode) .container {
                background: var(--container-bg);
                color: var(--text-color);
            }
        }
        
        @media (max-width: 480px) {
            body {
                padding: 0;
                background: var(--container-bg);
            }
            .container {
                max-width: 100%;
                margin: 0;
                padding: 20px;
                min-height: 100vh;
                border-radius: 0;
                box-shadow: none;
            }
            .background-shapes {
                display: none;
            }
            body::before,
            body::after {
                display: none;
            }
        }
        
        .footer {
            position: fixed;
            bottom: 20px;
            left: 50%;
            transform: translateX(-50%);
            color: var(--secondary-text);
            font-size: 12px;
            opacity: 0.9;
            transition: all 0.3s ease;
            white-space: nowrap;
            background: var(--container-bg);
            padding: 8px 16px;
            border-radius: var(--border-radius);
            box-shadow: var(--container-shadow);
            backdrop-filter: blur(8px);
            z-index: 2;
        }
        
        .footer:hover {
            opacity: 1;
            transform: translateX(-50%) translateY(-2px);
        }
        
        .footer a {
            color: var(--primary-color);
            text-decoration: none;
            font-weight: 500;
            transition: all 0.3s ease;
        }
        
        .footer a:hover {
            text-decoration: underline;
            opacity: 0.9;
        }
        
        .language-switcher {
            margin-top: 24px;
            text-align: center;
            display: flex;
            justify-content: center;
            gap: 8px;
        }
        
        .language-switcher button {
            background: none;
            border: 2px solid transparent;
            padding: 6px 12px;
            color: var(--secondary-text);
            font-size: 14px;
            cursor: pointer;
            transition: all 0.2s ease;
            border-radius: var(--border-radius-sm);
        }
        
        .language-switcher button:hover {
            background: rgba(0, 122, 255, 0.1);
            color: var(--primary-color);
        }
        
        .language-switcher button[data-active="true"] {
            border-color: var(--primary-color);
            color: var(--primary-color);
        }
        
        .theme-switcher {
            position: absolute;
            top: 20px;
            right: 20px;
            z-index: 2;
        }
        
        .theme-switcher button {
            background: none;
            border: none;
            color: var(--secondary-text);
            font-size: 18px;
            cursor: pointer;
            padding: 8px;
            border-radius: var(--border-radius-sm);
            transition: all 0.3s ease;
            width: 36px;
            height: 36px;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        
        .theme-switcher button i {
            width: 18px;
            text-align: center;
        }
        
        .theme-switcher button:hover {
            background: rgba(0, 122, 255, 0.1);
            color: var(--primary-color);
        }
    </style>
    <script>
        (function() {
            const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
            const savedTheme = localStorage.getItem('theme-preference');

            if (savedTheme) {
                document.documentElement.classList.add(`${savedTheme}-mode`);
            } else if (prefersDark) {
                document.documentElement.classList.add('dark-mode');
            }
        })();
    </script>
</head>
<body>
    <div class="background-shapes">
        <div class="shape shape-1"></div>
        <div class="shape shape-2"></div>
        <div class="shape shape-3"></div>
        <div class="shape shape-4"></div>
    </div>
    <div class="container">
        <div class="theme-switcher">
            <button onclick="toggleTheme()" id="themeToggle">
                <i class="fas fa-moon"></i>
            </button>
        </div>
        <div class="error-code">404</div>
        <h1 class="error-message" data-i18n="error-title">Page Not Found</h1>
        <p class="error-description" data-i18n="error-description">
            The page you are looking for might have been removed, had its name changed, or is temporarily unavailable.
        </p>
        <a href="/s/login" class="back-button" data-i18n="back-button">Back to Login</a>
        <div class="language-switcher">
            <button onclick="changeLanguage('en')" data-lang="en">EN</button>
            <button onclick="changeLanguage('zh-TW')" data-lang="zh-TW">繁</button>
            <button onclick="changeLanguage('zh-CN')" data-lang="zh-CN">简</button>
        </div>
    </div>

    <div class="footer">
        Powered by <a href="https://github.com/OG-Open-Source" target="_blank">OG Open Source</a>
    </div>

    <script>
        const i18n = {
            'en': {
                'login-prompt': 'Please login to continue',
                'username': 'Username',
                'password': 'Password',
                'login-button': 'Login',
                'logging-in': 'Logging in...',
                'error-invalid': 'Invalid username or password',
                'error-server': 'Server error, please try again later',
                'error-invalid-chars': 'Invalid characters detected'
            },
            'zh-TW': {
                'login-prompt': '請登入以繼續',
                'username': '用戶名',
                'password': '密碼',
                'login-button': '登入',
                'logging-in': '登入中...',
                'error-invalid': '用戶名或密碼錯誤',
                'error-server': '發生錯誤，請稍後再試',
                'error-invalid-chars': '包含無效字符'
            },
            'zh-CN': {
                'login-prompt': '请登录以继续',
                'username': '用户名',
                'password': '密码',
                'login-button': '登录',
                'logging-in': '登录中...',
                'error-invalid': '用户名或密码错误',
                'error-server': '发生错误，请稍后再试',
                'error-invalid-chars': '包含无效字符'
            }
        };

        function getBrowserLanguage() {
            const lang = navigator.language || navigator.userLanguage;
            if (lang.startsWith('zh')) {
                return lang === 'zh-TW' ? 'zh-TW' : 'zh-CN';
            }
            return 'en';
        }

        let currentLanguage = localStorage.getItem('panelbase-language') || getBrowserLanguage();

        function updateTexts() {
            document.querySelectorAll('[data-i18n]').forEach(element => {
                const key = element.getAttribute('data-i18n');
                element.textContent = i18n[currentLanguage][key];
            });
            document.documentElement.lang = currentLanguage;

            document.querySelectorAll('[data-lang]').forEach(button => {
                button.setAttribute('data-active', button.getAttribute('data-lang') === currentLanguage);
            });
        }

        function changeLanguage(lang) {
            currentLanguage = lang;
            localStorage.setItem('panelbase-language', lang);
            updateTexts();
        }

        updateTexts();

        async function handleLogin(event) {
            event.preventDefault();
            const errorMessage = document.getElementById('errorMessage');
            const submitButton = event.target.querySelector('button[type="submit"]');
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;

            if (!/^[A-Za-z0-9]+$/.test(username)) {
                errorMessage.innerHTML = `<i class="fas fa-exclamation-circle"></i>${i18n[currentLanguage]['error-invalid-chars']}`;
                errorMessage.hidden = false;
                return;
            }

            if (!/^[A-Za-z0-9!@$]+$/.test(password)) {
                errorMessage.innerHTML = `<i class="fas fa-exclamation-circle"></i>${i18n[currentLanguage]['error-invalid-chars']}`;
                errorMessage.hidden = false;
                return;
            }

            try {
                submitButton.disabled = true;
                submitButton.querySelector('[data-i18n="login-button"]').textContent = i18n[currentLanguage]['logging-in'];

                const urlParams = new URLSearchParams(window.location.search);
                const redirectUrl = urlParams.get('redirect') || '/panel';

                const response = await fetch('/s/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: `username=${encodeURIComponent(username)}&password=${encodeURIComponent(password)}&redirect=${encodeURIComponent(redirectUrl)}`
                });

                const data = await response.json();

                if (data.status === 'success' && data.code === '200') {
                    window.location.href = data.redirect;
                } else {
                    let errorKey = 'error-invalid';
                    if (data.code === '400') {
                        errorKey = 'error-invalid-chars';
                    }
                    errorMessage.innerHTML = `<i class="fas fa-exclamation-circle"></i>${i18n[currentLanguage][errorKey]}`;
                    errorMessage.hidden = false;
                    submitButton.disabled = false;
                    submitButton.querySelector('[data-i18n="login-button"]').textContent = i18n[currentLanguage]['login-button'];
                }
            } catch (error) {
                console.error('Error:', error);
                errorMessage.innerHTML = `<i class="fas fa-exclamation-circle"></i>${i18n[currentLanguage]['error-server']}`;
                errorMessage.hidden = false;
                submitButton.disabled = false;
                submitButton.querySelector('[data-i18n="login-button"]').textContent = i18n[currentLanguage]['login-button'];
            }
        }

        function toggleTheme() {
            const html = document.documentElement;
            const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
            const isDark = html.classList.contains('dark-mode') || (!html.classList.contains('light-mode') && prefersDark);

            if (isDark) {
                html.classList.remove('dark-mode');
                html.classList.add('light-mode');
                document.querySelector('#themeToggle i').className = 'fas fa-moon';
                localStorage.setItem('theme-preference', 'light');
            } else {
                html.classList.remove('light-mode');
                html.classList.add('dark-mode');
                document.querySelector('#themeToggle i').className = 'fas fa-sun';
                localStorage.setItem('theme-preference', 'dark');
            }
        }

        document.addEventListener('DOMContentLoaded', function() {
            const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
            const savedTheme = localStorage.getItem('theme-preference');

            if (savedTheme) {
                document.documentElement.classList.add(`${savedTheme}-mode`);
                document.querySelector('#themeToggle i').className = `fas fa-${savedTheme === 'dark' ? 'sun' : 'moon'}`;
            } else if (prefersDark) {
                document.documentElement.classList.add('dark-mode');
                document.querySelector('#themeToggle i').className = 'fas fa-sun';
            }
        });
    </script>
</body>
</html>
ERROR_404_HTML
			;;
		esac
	fi
	exit 0
}

SHOW_FORBIDDEN() {
	local message="$1"
	SHOW_ERROR "403" "403" "$message"
}

SHOW_NOT_FOUND() {
	local message="$1"
	SHOW_ERROR "404" "404" "$message"
}

REDIRECT_TO_LOGIN() {
	local original_url="$1"
	local message="$2"
	[ -n "$message" ] && log_auth_event "INFO" "$message"

	if [ -n "$IS_CURL" ]; then
		SECURITY_HEADERS "application/json" "401"
		echo "{\"status\":\"error\",\"code\":\"401\",\"message\":\"Authentication required\"}"
	else
		echo "Content-type: text/html"
		echo "Status: 302"
		echo "Location: /s/login?redirect=$(urlencode "$original_url")"
		echo
	fi
	exit 0
}

urlencode() {
	local string="$1"
	echo -n "$string" | xxd -plain | tr -d '\n' | sed 's/\(..\)/%\1/g'
}

urldecode() {
	local encoded="$1"
	echo -n "$encoded" | sed 's/%/\\x/g' | xargs -0 printf "%b"
}

cleanup_sessions() {
	local current_time=$(date +%s)
	local temp_file=$(mktemp)

	# 清理過期的 session
	awk -F: -v time="$current_time" -v max_age="$SESSION_LIFETIME" \
		'(time - $3) < max_age {print $0}' "$SESSION_FILE" >"$temp_file"

	mv "$temp_file" "$SESSION_FILE"
	chmod 600 "$SESSION_FILE"
}

# 獲取請求信息
AUTH_TOKEN=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')
ORIGINAL_URL="$REQUEST_URI"
REFERER=$(echo "$HTTP_REFERER" | grep -oP 'http://[^/]+\K.*' || echo "")
IS_CURL=$(echo "$HTTP_USER_AGENT" | grep -i "curl")
PATH_INFO="${PATH_INFO:-/}"

# 清理過期的 session
cleanup_sessions

# 處理根路徑訪問
if [ "$ORIGINAL_URL" = "/" ]; then
	REDIRECT_TO_LOGIN "/" "Root path access"
	exit 0
fi

# 處理登入相關的請求
if [ "$PATH_INFO" = "/login" ]; then
	if [ "$REQUEST_METHOD" = "POST" ]; then
		read -n $CONTENT_LENGTH POST_DATA
		USERNAME=$(echo "$POST_DATA" | grep -oP 'username=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')
		PASSWORD=$(echo "$POST_DATA" | grep -oP 'password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')
		REDIRECT_URL=$(echo "$QUERY_STRING" | grep -oP 'redirect=\K[^&]+' | urldecode)

		if ! [[ "$USERNAME" =~ ^[A-Za-z0-9]+$ ]]; then
			SECURITY_HEADERS "application/json" "400"
			echo '{"status":"error","code":"400","message":"Invalid username format"}'
			exit 0
		fi

		if ! [[ "$PASSWORD" =~ ^[A-Za-z0-9!@$]+$ ]]; then
			SECURITY_HEADERS "application/json" "400"
			echo '{"status":"error","code":"400","message":"Invalid password format"}'
			exit 0
		fi

		STORED_HASH=$(grep "^$USERNAME:" "$CONFIG_FILE" | cut -d: -f2)
		INPUT_HASH=$(echo -n "$PASSWORD" | md5sum | cut -d' ' -f1)

		if [ "$STORED_HASH" = "$INPUT_HASH" ]; then
			current_time=$(date +%s)
			token=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)
			echo "$token:$USERNAME:$current_time" >>"$SESSION_FILE"
			chmod 600 "$SESSION_FILE"

			SECURITY_HEADERS "application/json" "200"
			echo "Set-Cookie: auth_token=$token; Path=/; HttpOnly; SameSite=Strict; Max-Age=$SESSION_LIFETIME"
			echo
			echo "{\"status\":\"success\",\"code\":\"200\",\"redirect\":\"$REDIRECT_URL\"}"
		else
			sleep 1
			SECURITY_HEADERS "application/json" "401"
			echo '{"status":"error","code":"401","message":"Invalid username or password"}'
		fi
	else
		# 顯示登入頁面
		SECURITY_HEADERS
		cat <<'LOGIN_HTML'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login - PanelBase</title>
    <link rel="icon" type="image/png" sizes="128x128" href="/favicon.png">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css" />
    <style>
        /* ... 保留原有的 style ... */
    </style>
</head>
<body>
    <div class="background-shapes">
        <div class="shape shape-1"></div>
        <div class="shape shape-2"></div>
        <div class="shape shape-3"></div>
        <div class="shape shape-4"></div>
    </div>
    <div class="container">
        <div class="header">
            <h1>PanelBase</h1>
            <p data-i18n="login-prompt">Please login to continue</p>
        </div>
        <form id="loginForm" onsubmit="handleLogin(event)">
            <div class="form-group">
                <label for="username" data-i18n="username">Username</label>
                <div class="input-container">
                    <i class="fas fa-user"></i>
                    <input type="text" id="username" name="username" required autocomplete="username" pattern="[A-Za-z0-9]+" title="Username" onkeypress="return /[A-Za-z0-9]/.test(event.key)">
                </div>
            </div>
            <div class="form-group">
                <label for="password" data-i18n="password">Password</label>
                <div class="input-container">
                    <i class="fas fa-lock"></i>
                    <input type="password" id="password" name="password" required autocomplete="current-password" pattern="[A-Za-z0-9!@$]+" title="Password" onkeypress="return /[A-Za-z0-9!@$]+/.test(event.key)">
                </div>
                <div id="errorMessage" hidden></div>
            </div>
            <button type="submit">
                <span data-i18n="login-button">Login</span>
            </button>
        </form>
        <div class="language-switcher">
            <button onclick="changeLanguage('en')" data-lang="en">EN</button>
            <button onclick="changeLanguage('zh-TW')" data-lang="zh-TW">繁</button>
            <button onclick="changeLanguage('zh-CN')" data-lang="zh-CN">简</button>
        </div>
        <div class="theme-switcher">
            <button onclick="toggleTheme()" id="themeToggle">
                <i class="fas fa-moon"></i>
            </button>
        </div>
    </div>

    <div class="footer">
        Powered by <a href="https://github.com/OG-Open-Source" target="_blank">OG Open Source</a>
    </div>

    <script>
        /* ... 保留原有的 script ... */
    </script>
</body>
</html>
LOGIN_HTML
	fi
	exit 0
fi

# 檢查是否需要驗證
is_public_resource() {
	local url="$1"
	case "$url" in
	"/s/login" | "/favicon.ico")
		return 0
		;;
	*)
		return 1
		;;
	esac
}

if is_public_resource "$ORIGINAL_URL"; then
	if [ "$ORIGINAL_URL" = "/favicon.ico" ]; then
		SECURITY_HEADERS "image/x-icon"
		cat "$DOCUMENT_ROOT/favicon.ico"
	fi
	exit 0
fi

# 驗證 session
if [ -z "$AUTH_TOKEN" ]; then
	log_auth_event "INFO" "No session token found"
	REDIRECT_TO_LOGIN "$ORIGINAL_URL" "No session"
	exit 0
fi

CURRENT_TIME=$(date +%s)
VALID_SESSION=$(awk -F: -v token="$AUTH_TOKEN" -v time="$CURRENT_TIME" -v max_age="$SESSION_LIFETIME" \
	'$1 == token && (time - $3) < max_age {print $2}' "$SESSION_FILE")

if [ -z "$VALID_SESSION" ]; then
	log_auth_event "WARN" "Invalid session token: $AUTH_TOKEN"
	echo "Set-Cookie: auth_token=; Path=/; HttpOnly; SameSite=Strict; Max-Age=0; Expires=Thu, 01 Jan 1970 00:00:00 GMT"
	REDIRECT_TO_LOGIN "$ORIGINAL_URL" "Invalid session"
	exit 0
fi

# 處理檔案請求
REQUESTED_FILE="${DOCUMENT_ROOT}${ORIGINAL_URL}"

if echo "$REQUESTED_FILE" | grep -q "\.\."; then
	SHOW_FORBIDDEN "Path traversal attempt detected"
fi

if ! check_file_access "$REQUESTED_FILE" "$REFERER"; then
	log_auth_event "WARN" "Access denied to file: $ORIGINAL_URL (Mode: $ACCESS_CONTROL_MODE, Referer: $REFERER)"
	SHOW_FORBIDDEN "Access to this resource is not allowed"
fi

if [ ! -f "$REQUESTED_FILE" ]; then
	log_auth_event "INFO" "404 Not Found: $REQUESTED_FILE"
	SHOW_NOT_FOUND "The requested URL $ORIGINAL_URL was not found on this server"
fi

# 處理檔案回應
EXTENSION="${REQUESTED_FILE##*.}"
case "$EXTENSION" in
"html")
	SECURITY_HEADERS
	;;
"css")
	SECURITY_HEADERS "text/css"
	echo "Cache-Control: public, max-age=$CACHE_MAX_AGE"
	;;
"js")
	SECURITY_HEADERS "application/javascript"
	echo "Cache-Control: public, max-age=$CACHE_MAX_AGE"
	;;
"png" | "jpg" | "jpeg" | "gif")
	SECURITY_HEADERS "image/${EXTENSION}"
	echo "Cache-Control: public, max-age=$CACHE_MAX_AGE"
	;;
"svg")
	SECURITY_HEADERS "image/svg+xml"
	echo "Cache-Control: public, max-age=$CACHE_MAX_AGE"
	;;
"woff" | "woff2" | "ttf" | "eot")
	SECURITY_HEADERS "font/${EXTENSION}"
	echo "Cache-Control: public, max-age=$CACHE_MAX_AGE"
	;;
*)
	SECURITY_HEADERS "application/octet-stream"
	;;
esac

cat "$REQUESTED_FILE"
