class SpanManager {
    constructor() {}

    create(key, value) {
        const colorMap = {
            interval: "#007bff",         // blue
            timeout: "#28a745",          // green
            enabled: "#17a2b8",          // teal
            onFail: "#ffc107",           // yellow
            role: "#dc3545",             // red
            depth: "#6f42c1",            // purple
            recordPending: "#fd7e14",    // orange
            description: "#6610f2",      // deep purple
            intervalOrigin: "#20c997",   // cyan
            requires: "#e83e8c",         // pink
            startDelay: "#fd7e14",       // orange
            recordPending: "#20c997",    // cyan
            other: "#6c757d"             // fallback grey
          };

        const span = document.createElement("span");
        span.setAttribute("data-role", key);
        span.textContent = value;
        span.className = "badge bg-secondary";
        span.style.padding = "2px 6px";
        span.style.background = colorMap[key] || colorMap["other"];
        span.style.color = "white";
        span.style.borderRadius = "4px";

        return span;
    }
}

// Expose globally
window.SpanManager = SpanManager;
