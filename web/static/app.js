function showToast(message, type = "success") {
    const existingToasts = document.querySelectorAll(".toast");
    existingToasts.forEach((t) => t.remove());

    const toast = document.createElement("div");
    toast.className = `toast toast-${type}`;
    toast.textContent = message;

    const container =
        document.querySelector(".toast-container") || createToastContainer();
    container.appendChild(toast);

    // Trigger reflow for animation
    void toast.offsetWidth;
    toast.classList.add("show");

    setTimeout(() => {
        toast.classList.remove("show");
        setTimeout(() => toast.remove(), 300);
    }, 4000);
}

function createToastContainer() {
    const container = document.createElement("div");
    container.className = "toast-container";
    document.body.appendChild(container);
    return container;
}

document.addEventListener("htmx:confirm", function (evt) {
    // The message is provided via the hx-confirm attribute
    const message = evt.detail.question;
    const dialog = document.getElementById("confirm-dialog");

    if (dialog && message) {
        // Prevent the default browser confirm
        evt.preventDefault();

        const messageEl = document.getElementById("confirm-message");
        const okBtn = document.getElementById("confirm-ok");
        const cancelBtn = document.getElementById("confirm-cancel");

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
document.addEventListener("showToast", function (evt) {
    const message = evt.detail.value || evt.detail.message;
    const type = evt.detail.type || "success";
    if (message) {
        showToast(message, type);
    }
});

function copyValuesToInputs(fieldValues) {
    Object.entries(fieldValues).forEach(([inputId, value]) => {
        const input = document.getElementById(inputId);
        if (!input || value === undefined || value === null || value === "") {
            return;
        }
        input.value = value;
    });
}

function copyHealthFromLastWeek(button) {
    copyValuesToInputs({
        sleep_score: button.dataset.sleepScore,
        waist_cm: button.dataset.waistCm,
        rhr: button.dataset.rhr,
        systolic_bp: button.dataset.systolicBp,
        diastolic_bp: button.dataset.diastolicBp,
        nutrition_score: button.dataset.nutritionScore,
    });
    showToast("Last week's health values copied into this week's form.");
}

function copyFitnessFromLastWeek(button) {
    copyValuesToInputs({
        vo2_max: button.dataset.vo2Max,
        workouts: button.dataset.workouts,
        daily_steps: button.dataset.dailySteps,
        mobility: button.dataset.mobility,
        cardio_recovery: button.dataset.cardioRecovery,
        leg_press_set: button.dataset.legPressSet,
    });
    showToast("Last week's fitness values copied into this week's form.");
}

function copyCognitionFromLastWeek(button) {
    copyValuesToInputs({
        mindfulness: button.dataset.mindfulness,
        deep_learning: button.dataset.deepLearning,
        stress_score: button.dataset.stressScore,
        social_days: button.dataset.socialDays,
    });
    showToast("Last week's cognition values copied into this week's form.");
}

function registerServiceWorker() {
    if ("serviceWorker" in navigator) {
        window.addEventListener("load", () => {
            navigator.serviceWorker
                .getRegistration("/sw.js")
                .then((registration) => {
                    if (!registration) {
                        navigator.serviceWorker
                            .register("/sw.js")
                            .then((reg) => {
                                console.log(
                                    "ServiceWorker registration successful with scope: ",
                                    reg.scope
                                );
                            })
                            .catch((err) => {
                                console.error(
                                    "ServiceWorker registration failed: ",
                                    err
                                );
                            });
                    } else {
                        console.log(
                            "ServiceWorker already registered with scope: ",
                            registration.scope
                        );
                    }
                });
        });
    }
}

// Auto-register on load
registerServiceWorker();
