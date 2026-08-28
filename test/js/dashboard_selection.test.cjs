const { test } = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');

test('selection survives reload and falls back when a saved cluster is absent', () => {
    let saved;

    const storage = { getItem: () => saved, setItem: (key, value) => { saved = value; } };
    const source = fs.readFileSync('static/js/dashboard.js', 'utf8').replace(/initializeDashboard\(\);\s*$/, '');
    function load(remoteExists) {
        const ctx = vm.createContext({ sessionStorage: storage, window: { addEventListener() {} } });
        vm.runInContext(source + '\nglobalThis.views = dashboardViews;', ctx);
        function view() {
            const classes = () => ({ toggle(name, value) { this[name] = value; } });
            const link = { setAttribute(name, value) { this[name] = value; } };
            return { tab: { classList: classes(), querySelector: () => link }, panel: { classList: classes() } };
        }
        ctx.views.set('local', view());
        if (remoteExists) ctx.views.set('remote:West Coast', view());
        return ctx;
    }
    load(true).rememberDashboardTab('remote:West Coast');
    const restored = load(true);
    restored.restoreDashboardTab();
    assert.equal(restored.views.get('remote:West Coast').panel.classList.active, true);
    assert.equal(restored.views.get('local').panel.classList.active, false);
    assert.equal(restored.views.get('remote:West Coast').tab.querySelector()['aria-selected'], 'true');
    restored.selectDashboardTab('local');
    assert.equal(restored.views.get('remote:West Coast').panel.classList.active, false);
    restored.rememberDashboardTab('remote:West Coast');
    const removed = load(false);
    removed.restoreDashboardTab();
    assert.equal(removed.views.get('local').panel.classList.active, true);
    assert.equal(saved, 'local');
});
