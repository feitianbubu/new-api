document.addEventListener('DOMContentLoaded', () => {
    fetch('/api/user/self')
        .catch(() => ({ ok: false }))
        .then(r => r.ok || (window.location.href = '/login'));
});