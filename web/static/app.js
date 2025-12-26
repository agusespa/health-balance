function showToast(message, type = 'success') {
    const existingToasts = document.querySelectorAll('.toast');
    existingToasts.forEach(t => t.remove());

    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;

    const container = document.querySelector('.toast-container') || createToastContainer();
    container.appendChild(toast);

    // Trigger reflow for animation
    void toast.offsetWidth;
    toast.classList.add('show');

    setTimeout(() => {
        toast.classList.remove('show');
        setTimeout(() => toast.remove(), 300);
    }, 4000);
}

function createToastContainer() {
    const container = document.createElement('div');
    container.className = 'toast-container';
    document.body.appendChild(container);
    return container;
}

document.addEventListener('htmx:confirm', function (evt) {
    // The message is provided via the hx-confirm attribute
    const message = evt.detail.question;
    const dialog = document.getElementById('confirm-dialog');

    if (dialog && message) {
        // Prevent the default browser confirm
        evt.preventDefault();

        const messageEl = document.getElementById('confirm-message');
        const okBtn = document.getElementById('confirm-ok');
        const cancelBtn = document.getElementById('confirm-cancel');

        messageEl.textContent = message;

        // Function to handle the result
        const handleResult = (confirmed) => {
            dialog.onclose = null; // Prevent recursion from dialog.close()
            dialog.close();
            if (confirmed) {
                // This resumes the HTMX request
                evt.detail.issueRequest(true);
            }
            // Cleanup listeners
            okBtn.onclick = null;
            cancelBtn.onclick = null;
        };

        okBtn.onclick = () => handleResult(true);
        cancelBtn.onclick = () => handleResult(false);

        // Also handle escape key / clicking outside
        dialog.onclose = () => handleResult(false);

        dialog.showModal();
    }
});

// Global listener for showToast events (can be triggered from backend via HX-Trigger)
document.addEventListener('showToast', function (evt) {
    const message = evt.detail.value || evt.detail.message;
    const type = evt.detail.type || 'success';
    if (message) {
        showToast(message, type);
    }
});
