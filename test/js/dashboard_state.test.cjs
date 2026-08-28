const { test } = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');

function context() {
    const ctx = vm.createContext({ console, setTimeout, clearTimeout });
    vm.runInContext(fs.readFileSync('static/js/classes/DashboardClusterView.js', 'utf8') + '\nglobalThis.View = DashboardClusterView;', ctx);
    return ctx;
}

function cell(dataset) {
    return { dataset, classList: { toggle(name, value) { this[name] = value; } } };
}

test('two cluster views keep search and node filters independent', () => {
    const { View } = context();
    function makeView() {
        const view = Object.create(View.prototype);
        view.nodes = [{ online: true }, { online: false }];
        view.nodeFilter = 'show_all';
        view.searchText = '';
        view.rows = [cell({ resourceName: 'Database' }), cell({ resourceName: 'Web' })];
        view.cells = [cell({ nodeIndex: 0 }), cell({ nodeIndex: 1 })];
        view.tbody = { querySelectorAll: selector => selector.includes('resource') ? view.rows : view.cells };
        return view;
    }
    const first = makeView(), second = makeView();
    first.setSearch(' DATA ');
    first.setNodeFilter('online');
    second.applyFilters();
    assert.deepEqual(first.rows.map(row => row.classList.hidden), [false, true]);
    assert.deepEqual(first.cells.map(row => row.classList.hidden), [false, true]);
    assert.deepEqual(second.rows.map(row => row.classList.hidden), [false, false]);
    assert.deepEqual(second.cells.map(row => row.classList.hidden), [false, false]);
    first.setSearch('');
    assert.deepEqual(first.rows.map(row => row.classList.hidden), [false, false]);
});

test('dashboard polling uses its own epoch and stops on cancellation', async () => {
    const controller = new AbortController();
    let url;
    const ctx = context();
    ctx.fetch = async value => { url = value; return { ok: true, json: async () => ({ epoch: 'new' }) }; };
    // Access to shared sessionStorage would fail in this context.
    vm.runInContext(fs.readFileSync('static/js/common.js', 'utf8'), ctx);
    let calls = 0;
    await ctx.pollClusterStatus(async () => { calls++; controller.abort(); }, {
        signal: controller.signal, getEpoch: () => 'cluster-specific', monitorURL: '/monitor',
    });
    assert.equal(url, '/monitor?cluster-specific');
    assert.equal(calls, 1);
});

test('a response arriving after cancellation does not render', async () => {
    const { View } = context();
    const controller = new AbortController();
    const view = Object.create(View.prototype);
    view.client = { fetchStatus: async () => { controller.abort(); return {}; } };
    view.render = () => assert.fail('cancelled view rendered');
    await view.refresh(controller.signal);
});

test('initial fetch failure retries status before starting a monitor', async () => {
    const ctx = context();
    ctx.AbortController = AbortController;
    ctx.DashboardAuthenticationError = class extends Error {};
    const view = Object.create(ctx.View.prototype);
    view.config = { local: true };
    view.controller = null;
    const calls = [];
    view.refresh = async signal => {
        calls.push('status');
        if (calls.length === 1) throw new Error('temporary failure');
        view.stop();
    };
    view.client = { monitor: async () => assert.fail('monitored before status recovery') };
    view.showError = () => {};
    view.showRetry = () => {};
    view.wait = async () => {};
    await view.start();
    assert.deepEqual(calls, ['status', 'status']);
});

test('initial remote connection failure offers login without an error banner', async () => {
    const ctx = context();
    ctx.AbortController = AbortController;
    ctx.DashboardAuthenticationError = class extends Error {};
    const view = Object.create(ctx.View.prototype);
    view.config = { name: 'remote' };
    view.client = {};
    view.clusterDetails = null;
    const failure = new Error('Cannot reach cluster');
    view.refresh = async () => { throw failure; };
    view.showError = () => {};
    view.showRetry = () => {};
    let offered = false;
    view.waitForLogin = async (signal, error) => {
        offered = true;
        assert.equal(error, undefined);
        view.stop();
    };
    await view.start();
    assert.equal(offered, true);
});

test('remote connection with submitted credentials retries automatically', async () => {
    const ctx = context();
    ctx.AbortController = AbortController;
    ctx.DashboardAuthenticationError = class extends Error {};
    const view = Object.create(ctx.View.prototype);
    view.config = { name: 'remote' };
    view.client = { tryNextHost: () => false };
    view.clusterDetails = { epoch: '1' };
    view.credentials = { username: 'hacluster', password: 'example' };
    view.refresh = async () => { throw Object.assign(new Error('Network failure'), { connectionFailure: true }); };
    view.showError = () => {};
    view.showRetry = () => {};
    view.waitForLogin = () => assert.fail('unexpected login prompt');
    let retried = false;
    view.wait = async () => { retried = true; view.stop(); };
    await view.start();
    assert.equal(retried, true);
});

test('failed login retries saved credentials and refreshes after success', async () => {
    const ctx = context();
    ctx.AbortController = AbortController;
    const view = Object.create(ctx.View.prototype);
    view.config = { name: 'remote' };
    const calls = [];
    let attempts = 0;
    view.client = {
        login: async (username, password) => {
            assert.equal(username, 'hacluster');
            assert.equal(password, 'example');
            calls.push('login');
            if (++attempts === 1) throw new Error('Denied');
        },
        tryNextHost: () => { calls.push('failover'); },
    };
    view.indicator = {};
    view.waitForLogin = async () => { view.credentials = { username: 'hacluster', password: 'example' }; };
    view.showRetry = () => calls.push('retry');
    view.wait = async ms => assert.equal(ms, 5000);
    view.refresh = async () => { calls.push('status'); view.stop(); };
    await view.start(true);
    assert.deepEqual(calls, ['login', 'retry', 'failover', 'login', 'status']);
    assert.equal(view.credentials, null);
});
