// Shared Auth & Validation Logic for Boligaksjonen

(function() {
    // Helper for debouncing
    function debounce(func, timeout = 300) {
        let timer;
        return (...args) => {
            clearTimeout(timer);
            timer = setTimeout(() => { func.apply(this, args); }, timeout);
        };
    }

    window.calculateEntropy = function(password) {
        if (!password) return 0;
        let poolSize = 0;
        if (/[a-z]/.test(password)) poolSize += 26;
        if (/[A-Z]/.test(password)) poolSize += 26;
        if (/[0-9]/.test(password)) poolSize += 10;
        if (/[^a-zA-Z0-9]/.test(password)) poolSize += 33;
        if (poolSize === 0) return 0;
        return password.length * Math.log2(poolSize);
    }

    window.checkPasswordStrength = function(password) {
        const entropy = window.calculateEntropy(password);
        const bar = document.getElementById('strength_bar');
        const text = document.getElementById('strength_text');
        
        const percent = Math.min((entropy / 100) * 100, 100);
        if (bar) bar.style.width = percent + '%';
        
        if (text && bar) {
            if (entropy < 40) {
                bar.className = 'h-full bg-red-500 transition-all duration-500';
                text.innerText = 'Veldig svakt';
                text.className = 'text-[9px] font-black uppercase tracking-widest mt-1 text-red-500';
            } else if (entropy < 60) {
                bar.className = 'h-full bg-yellow-500 transition-all duration-500';
                text.innerText = 'Middels';
                text.className = 'text-[9px] font-black uppercase tracking-widest mt-1 text-yellow-500';
            } else if (entropy < 100) {

                bar.className = 'h-full bg-green-500 transition-all duration-500';
                text.innerText = 'Sterkt';
                text.className = 'text-[9px] font-black uppercase tracking-widest mt-1 text-green-500';
            } else {
                bar.className = 'h-full bg-blue-500 transition-all duration-500 shadow-[0_0_10px_rgba(59,130,246,0.5)]';
                text.innerText = 'Veldig sterkt';
                text.className = 'text-[9px] font-black uppercase tracking-widest mt-1 text-blue-500';
            }
        }

        window.validateSignupForm();
    }

    window.validateSignupForm = function() {
        const passEl = document.getElementById('signup_password');
        const confirmEl = document.getElementById('signup_confirm');
        const noticeEl = document.getElementById('username_notice');
        const submit = document.getElementById('signup_submit');
        
        if (!passEl || !confirmEl || !submit) return;

        const pass = passEl.value;
        const confirm = confirmEl.value;
        const entropy = window.calculateEntropy(pass);
        
        const isMatch = pass === confirm && pass !== "";
        const isStrong = entropy >= 40;
        const isUsernameAvailable = noticeEl && noticeEl.getAttribute('data-available') === 'true';

        if (confirm !== "" && !isMatch) {
            confirmEl.classList.add('border-red-500');
            confirmEl.classList.remove('border-gray-200');
        } else if (isMatch) {
            confirmEl.classList.add('border-green-500');
            confirmEl.classList.remove('border-red-500', 'border-gray-200');
        } else {
            confirmEl.classList.remove('border-red-500', 'border-green-500');
            confirmEl.classList.add('border-gray-200');
        }

        if (isMatch && isStrong && isUsernameAvailable) {
            submit.disabled = false;
            submit.className = 'flex-1 py-3 bg-blue-600 hover:bg-blue-700 text-white font-black uppercase text-[10px] sm:text-xs rounded-2xl shadow-xl transition-all';
        } else {
            submit.disabled = true;
            submit.className = 'flex-1 py-3 bg-gray-200 cursor-not-allowed text-white font-black uppercase text-[10px] sm:text-xs rounded-2xl shadow-xl transition-all';
        }
    }

    // Username check is now handled by htmx in components/auth_dialogs.templ


    window.validateSignup = function(e) {
        const pass = document.getElementById('signup_password').value;
        const confirm = document.getElementById('signup_confirm').value;
        
        if (pass !== confirm) {
            alert("Passordene er ikke like!");
            e.preventDefault();
            return false;
        }
        
        const entropy = window.calculateEntropy(pass);
        if (entropy < 40) {
            alert("Passordet er for svakt.");
            e.preventDefault();
            return false;
        }
        return true;
    }

    window.showSignup = function() {
        const loginDialog = document.getElementById('loginDialog');
        const signupDialog = document.getElementById('signupDialog');
        if (loginDialog) loginDialog.close();
        if (signupDialog) signupDialog.showModal();
    }

    window.showLogin = function() {
        const loginDialog = document.getElementById('loginDialog');
        const signupDialog = document.getElementById('signupDialog');
        if (signupDialog) signupDialog.close();
        if (loginDialog) loginDialog.showModal();
    }
})();
