const { test } = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');

function setup(fetch) {
    const context = vm.createContext({ fetch, URL, URLSearchParams, AbortController, setTimeout, clearTimeout,
        window: { location: { origin: 'https://local:7630' } } });
    vm.runInContext(fs.readFileSync('static/js/classes/DashboardClusterClient.js', 'utf8') +
        '\nglobalThis.Client = DashboardClusterClient; globalThis.AuthError = DashboardAuthenticationError;', context);
    return context;
}
function response(data, status = 200, type = 'application/json') {
    return { ok: status >= 200 && status < 300, status, headers: { get: () => type },
        json: async () => data, text: async () => data };
}
const config = { name: 'West', host: 'node1', https: true, port: 7630 };
const legacy = { meta: { status: 'ok', epoch: '1:2:3', dc: 'node1' }, crm_config: { cluster_name: 'West' },
    nodes: [{ name: 'node1', state: 'online' }, { name: 'node2', state: 'offline' }],
    resources: [{ id: 'database', state: 'master', running_on: { node1: 'master' } }] };

test('Go status uses cookies and retains protocol for monitoring', async () => {
    const calls = [];
    const { Client } = setup(async (url, opts) => {
        calls.push({ url, opts });
        return response(url.includes('/monitor') ? { epoch: '1:2:3' } : { clusterDetails: { epoch: '1:2:3' }, nodes: [], resources: [] });
    });
    const client = new Client(config);
    await client.fetchStatus();
    assert.equal(client.protocol, 'go');
    assert.equal(await client.monitor('1:2:3'), '1:2:3');
    assert.equal(calls[0].opts.method, 'POST');
    assert.equal(calls[0].opts.credentials, 'include');
    assert.equal(calls[1].url, 'https://node1:7630/monitor?1:2:3');
});

test('legacy fallback normalizes roles and does not reprobe Go', async () => {
    const calls = [];
    const { Client } = setup(async url => {
        calls.push(url);
        return url.includes('/api/') ? response({}, 404) : url.includes('/monitor') ? response({ epoch: '1:2:3' }) : response(legacy);
    });
    const client = new Client(config);
    const data = await client.fetchStatus();
    assert.deepEqual(Array.from(data.resources[0].roles), ['promoted', 'offline']);
    assert.equal(data.nodes[0].isDC, true);
    await client.fetchStatus();
    await client.monitor('1:2:3');
    assert.equal(calls.filter(url => url.includes('/api/')).length, 1);
    assert.equal(calls.at(-1), 'https://node1:7630/monitor.json?1:2:3');
});

test('session expiry is distinct from connection errors', async () => {
    const { Client, AuthError } = setup(async () => response({}, 401));
    await assert.rejects(new Client(config).fetchStatus(), AuthError);
    const failed = setup(async () => { throw new TypeError('network'); });
    await assert.rejects(new failed.Client(config).fetchStatus(), error => !(error instanceof failed.AuthError));
});

test('login uses form encoding, includes cookies, and never stores the password', async () => {
    let request;
    const { Client } = setup(async (url, options) => { request = { url, options }; return response({}); });
    const client = new Client(config);
    await client.login('hacluster', 'test & password');
    assert.equal(request.url, 'https://node1:7630/login.json');
    assert.equal(request.options.body.get('session[password]'), 'test & password');
    assert.equal(request.options.credentials, 'include');
    assert.ok(!JSON.stringify(client).includes('test & password'));
});

test('plain Go offline responses render as offline, not JSON errors', async () => {
    const { Client } = setup(async () => response('Pacemaker cluster is offline', 503, 'text/plain'));
    const data = await new Client(config).fetchStatus();
    assert.equal(data.clusterDetails.status, 'offline');
    assert.equal(data.nodes.length, 0);
});

test('IPv6 addresses work and hostname URL injection is rejected', () => {
    const { Client } = setup();
    assert.equal(Client.clusterURL({ ...config, host: '::1' }), 'https://[::1]:7630');
    for (const host of ['user@host', 'host/path', 'host?query', 'host#fragment']) {
        assert.throws(() => Client.clusterURL({ ...config, host }));
    }
});

test('cancelling a pending connection aborts its fetch', async () => {
    const { Client } = setup((url, { signal }) => new Promise((resolve, reject) => {
        signal.addEventListener('abort', () => reject(new Error('aborted')), { once: true });
    }));
    const controller = new AbortController();
    const pending = new Client(config).fetchStatus(controller.signal);
    controller.abort();
    await assert.rejects(pending, /aborted/);
});

test('failover rotates discovered nodes without rewriting saved configuration', () => {
    const { Client } = setup();
    const client = new Client(config);
    client.rememberHosts([{ name: 'node1' }, { name: 'node2' }]);
    client.protocol = 'go';
    assert.equal(client.tryNextHost(), true);
    assert.equal(client.baseURL, 'https://node2:7630');
    assert.equal(client.config.host, 'node1');
    assert.equal(client.protocol, null);
    client.tryNextHost();
    assert.equal(client.baseURL, 'https://node1:7630');
});

test('legacy info bar shows messages without error metadata', () => {
    const { Client } = setup();
    const message = 'STONITH is disabled. For normal cluster operation, STONITH is required.';
    const data = Client.normalizeLegacy({ ...legacy, errors: [
        { link: '/cib/live/crm_config/edit', msg: message, type: 'warning', warning: true },
        'Another message',
        { type: 'warning' },
        null,
    ] });
    assert.equal(data.clusterDetails.summary, `${message}\nAnother message`);
});
