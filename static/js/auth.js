class AuthManager {
    constructor() {
        this.loginForm = document.getElementById('loginForm');
        this.registerForm = document.getElementById('registerForm');
        this.setupEventListeners();
    }

    setupEventListeners() {
        if (this.loginForm) {
            this.loginForm.addEventListener('submit', (e) => this.handleLogin(e));
        }
        if (this.registerForm) {
            this.registerForm.addEventListener('submit', (e) => this.handleRegister(e));
            const passwordInput = document.getElementById('password');
            if (passwordInput) {
                passwordInput.addEventListener('input', (e) => this.checkPasswordStrength(e.target.value));
            }
        }
    }

    async handleLogin(e) {
        e.preventDefault();
        const submitButton = e.target.querySelector('button[type="submit"]');
        submitButton.classList.add('btn-loading');

        const formData = new FormData(e.target);
        const credentials = {
            email: formData.get('email'),
            password: formData.get('password'),
            userType: formData.get('userType')
        };

        try {
            const response = await fetch('/api/auth/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(credentials)
            });

            const data = await response.json();

            if (data.success) {
                // Store token
                localStorage.setItem('token', data.token);
                localStorage.setItem('userType', data.userType);

                // Redirect based on user type
                switch (data.userType) {
                    case 'parent':
                        window.location.href = '/parent/dashboard';
                        break;
                    case 'driver':
                        window.location.href = '/driver/dashboard';
                        break;
                    case 'admin':
                        window.location.href = '/admin/dashboard';
                        break;
                }
            } else {
                this.showError('loginError', data.message || 'Invalid credentials');
            }
        } catch (error) {
            this.showError('loginError', 'An error occurred. Please try again.');
        } finally {
            submitButton.classList.remove('btn-loading');
        }
    }

    async handleRegister(e) {
        e.preventDefault();
        const submitButton = e.target.querySelector('button[type="submit"]');
        submitButton.classList.add('btn-loading');

        const formData = new FormData(e.target);
        
        // Validate password match
        if (formData.get('password') !== formData.get('confirmPassword')) {
            this.showError('registerError', 'Passwords do not match');
            submitButton.classList.remove('btn-loading');
            return;
        }

        const userData = {
            firstName: formData.get('firstName'),
            lastName: formData.get('lastName'),
            email: formData.get('email'),
            phone: formData.get('phone'),
            password: formData.get('password'),
            userType: formData.get('userType')
        };

        try {
            const response = await fetch('/api/auth/register', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(userData)
            });

            const data = await response.json();

            if (data.success) {
                // Store token and redirect to dashboard
                localStorage.setItem('token', data.token);
                localStorage.setItem('userType', data.userType);

                // Show success message and redirect
                alert('Registration successful! Please wait while we redirect you.');
                window.location.href = `/${data.userType}/dashboard`;
            } else {
                this.showError('registerError', data.message || 'Registration failed');
            }
        } catch (error) {
            this.showError('registerError', 'An error occurred. Please try again.');
        } finally {
            submitButton.classList.remove('btn-loading');
        }
    }

    checkPasswordStrength(password) {
        const strengthIndicator = document.createElement('div');
        strengthIndicator.className = 'password-strength';
        
        const strengthBar = document.createElement('div');
        
        // Check password strength
        let strength = 0;
        if (password.length >= 8) strength++;
        if (password.match(/[a-z]/) && password.match(/[A-Z]/)) strength++;
        if (password.match(/[0-9]/)) strength++;
        if (password.match(/[^a-zA-Z0-9]/)) strength++;

        // Update strength indicator
        switch (strength) {
            case 0:
            case 1:
                strengthBar.className = 'strength-weak';
                break;
            case 2:
            case 3:
                strengthBar.className = 'strength-medium';
                break;
            case 4:
                strengthBar.className = 'strength-strong';
                break;
        }

        strengthIndicator.appendChild(strengthBar);

        // Update or create strength indicator
        const existingIndicator = document.querySelector('.password-strength');
        if (existingIndicator) {
            existingIndicator.replaceWith(strengthIndicator);
        } else {
            document.getElementById('password').parentNode.appendChild(strengthIndicator);
        }
    }

    showError(elementId, message) {
        const errorElement = document.getElementById(elementId);
        if (errorElement) {
            errorElement.textContent = message;
            errorElement.style.display = 'block';
        }
    }
}

// Initialize authentication manager
const authManager = new AuthManager();
