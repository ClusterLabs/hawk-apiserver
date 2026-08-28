const { test } = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');

function fixture(response, active = true, confirmed = true) {
    let destroyed = false;
    let error;
    let request;
    let flash;
    const classList = () => ({ active: false, add(value) { this[value] = true; }, toggle(name, value) { this[name] = value; } });
    const local = { tab: { classList: classList(), querySelector: () => ({ setAttribute() {} }) }, panel: { classList: classList() } };
    const view = { config: { name: 'remote' }, tab: { classList: { contains: () => active } },
        destroy: () => { destroyed = true; }, showError: value => { error = value; } };
    const button = { disabled: false, closest: () => ({ dashboardView: view }) };
    const ctx = vm.createContext({ document: {
        createElement(tag) { return { tag, setAttribute(name, value) { this[name] = value; } }; },
        getElementById(id) { assert.equal(id, 'status-alert'); return { replaceWith(value) { flash = value; } }; },
    }, RemoveClusterPopup: class { result = Promise.resolve(confirmed); }, window: { addEventListener() {} }, fetch: async (url, options) => {
        request = { url, options }; return response;
    } });
    const source = fs.readFileSync('static/js/dashboard.js', 'utf8').replace(/initializeDashboard\(\);\s*$/, '');
    vm.runInContext(source + '\nglobalThis.views = dashboardViews;', ctx);
    ctx.views.set('local', local);
    ctx.views.set('remote:remote', view);
    return { ctx, button, local, get flash() { return flash; }, get destroyed() { return destroyed; }, get error() { return error; }, get request() { return request; } };
}

test('removing active remote tab deletes saved entry, destroys view and selects local', async () => {
    const f = fixture({ ok: true, status: 204 });
    await f.ctx.removeDashboardCluster(f.button);
    assert.equal(f.request.options.method, 'DELETE');
    assert.deepEqual(JSON.parse(f.request.options.body), { name: 'remote' });
    assert.equal(f.destroyed, true);
    assert.equal(f.ctx.views.has('remote:remote'), false);
    assert.equal(f.local.tab.classList.active, true);
    assert.equal(f.local.panel.classList.active, true);
    assert.equal(f.flash.tag, 'x-alert');
    assert.equal(f.flash.type, 'success');
    assert.equal(f.flash.message, 'Cluster removed successfully.');
    assert.equal(f.flash.visible, 'true');
});

test('failed delete retains view and enables retry', async () => {
    const f = fixture({ ok: false, status: 500 });
    await f.ctx.removeDashboardCluster(f.button);
    assert.equal(f.destroyed, false);
    assert.equal(f.flash, undefined);
    assert.equal(f.ctx.views.has('remote:remote'), true);
    assert.equal(f.button.disabled, false);
    assert.ok(f.error);
});

test('removing an inactive tab does not change active selection', async () => {
    const f = fixture({ ok: true, status: 204 }, false);
    await f.ctx.removeDashboardCluster(f.button);
    assert.equal(f.local.tab.classList.active, false);
});


test('cancelling removal keeps the saved cluster and tab without a request', async () => {
    const f = fixture({ ok: true, status: 204 }, true, false);
    await f.ctx.removeDashboardCluster(f.button);
    assert.equal(f.request, undefined);
    assert.equal(f.destroyed, false);
    assert.equal(f.flash, undefined);
    assert.equal(f.ctx.views.has('remote:remote'), true);
    assert.equal(f.button.disabled, false);
});
